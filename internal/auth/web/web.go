package web

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/astaxie/beego/session"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/chremoas/chremoas-ng/internal/auth"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/payloads"
	"github.com/dimfeld/httptreemux"
	"github.com/gregjones/httpcache"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	name = "auth"
)

//go:embed static/* templates/*
var content embed.FS

var (
	globalSessions *session.Manager
	apiClient      *goesi.APIClient
	authenticator  *goesi.SSOAuthenticator
)

type ResultModel struct {
	Title      string
	Auth       string
	DiscordUrl string
	Name       string
}

type Web struct {
	dependencies common.Dependencies
	templates    *template.Template
	ctx          context.Context
}

func New(ctx context.Context, deps common.Dependencies) (*Web, error) {
	ctx, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	// Setup our required globals.
	globalSessions, _ = session.NewManager("memory", &session.ManagerConfig{CookieName: "gosessionid", EnableSetCookie: true, Gclifetime: 600})
	go globalSessions.GC()

	httpClient := httpcache.NewMemoryCacheTransport().Client()

	// Get the ESI API Client
	apiClient = goesi.NewAPIClient(httpClient, "aba-auth-web maurer.it@gmail.com https://github.com/chremoas/auth-web")

	fullCallbackUrl := viper.GetString("oauth.callBackProtocol") + "://" + viper.GetString("oauth.callBackHost") + viper.GetString("oauth.callBackUrl")

	// Allocate an SSO Authenticator
	authenticator = goesi.NewSSOAuthenticator(
		httpClient,
		viper.GetString("oauth.clientId"),
		viper.GetString("oauth.clientSecret"),
		fullCallbackUrl,
		nil,
	)

	// Initialize my templates
	templates := template.New("auth-web")
	_, err := templates.ParseFS(content, "templates/*.html")
	if err != nil {
		sp.Error("Error parsing templates", zap.Error(err))
		return nil, err
	}

	return &Web{dependencies: deps, templates: templates, ctx: ctx}, nil
}

func (web Web) Auth(ctx context.Context) *httptreemux.ContextMux {
	mux := httptreemux.NewContextMux()

	mux.Handle(http.MethodGet, "/ready", addLoggerMiddleware(web.ctx, web.readiness))
	mux.Handle(http.MethodGet, "/static/*path", addLoggerMiddleware(web.ctx, web.serveFiles))
	mux.Handle(http.MethodGet, "/", addLoggerMiddleware(web.ctx, middleware(web.handleIndex)))
	mux.Handle(http.MethodGet, "/login", addLoggerMiddleware(web.ctx, middleware(web.handleEveLogin)))
	mux.Handle(http.MethodGet, viper.GetString("oauth.callBackUrl"), addLoggerMiddleware(web.ctx, middleware(web.handleEveCallback)))

	return mux
}

func (web Web) serveFiles(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.FS(content)).ServeHTTP(w, r)
}

func (web Web) readiness(w http.ResponseWriter, _ *http.Request) {
	_, sp := sl.OpenSpan(web.ctx)
	defer sp.Close()

	status := struct {
		Status string
	}{
		Status: "OK",
	}

	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		sp.Error("Error encoding status", zap.Error(err))
		http.Error(w, "Error encoding status", http.StatusInternalServerError)
	}
}

func (web Web) handleIndex(w http.ResponseWriter, _ *http.Request) {
	_, sp := sl.OpenSpan(web.ctx)
	defer sp.Close()

	err := web.templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		sp.Error("Error executing index", zap.Error(err))
		http.Error(w, "Error executing index", http.StatusInternalServerError)
	}
}

func (web Web) handleEveLogin(w http.ResponseWriter, r *http.Request) {
	_, sp := sl.OpenSpan(web.ctx)
	defer sp.Close()

	// Get the users session
	sess, _ := globalSessions.SessionStart(w, r)

	// Get the authenticator from the request context
	ssoauth := authenticatorFromContext(r.Context())

	// Generate a random 16 byte state.
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		sp.Error("Error generating random string", zap.Error(err))
		http.Error(w, "Error generating random string", http.StatusInternalServerError)
		return
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Save the state to the session to validate with the response.
	err = sess.Set("state", state)
	if err != nil {
		sp.Error("Error setting state", zap.Error(err))
		http.Error(w, "Error setting state", http.StatusInternalServerError)
		return
	}

	// Build the authorize URL
	// TODO: This is where we'd set extra needed scopes
	redirectUrl := ssoauth.AuthorizeURL(state, true, nil)

	// Redirect the user to CCP SSO
	http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
}

func (web Web) handleEveCallback(w http.ResponseWriter, r *http.Request) {
	_, sp := sl.OpenSpan(web.ctx)
	defer sp.Close()

	sess, _ := globalSessions.SessionStart(w, r)
	if sess == nil {
		sp.Info("No session, redirecting to /")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	internalAuthCode, err := web.doAuth(w, r, sess)
	if err != nil {
		// TODO: Make another template for errors specifically for this endpoint
		sp.Error("received an error from doAuth", zap.Error(err))
		http.Error(w, "Unknown auth error", http.StatusInternalServerError)
		return
	}

	err = web.templates.ExecuteTemplate(w, "authd.html",
		&ResultModel{
			Title:      "Authd Up",
			Auth:       *internalAuthCode,
			DiscordUrl: viper.GetString("discord.inviteUrl"),
			Name:       name,
		},
	)
	if err != nil {
		sp.Error("Error executing auth template", zap.Error(err))
		http.Error(w, "Error executing auth template", http.StatusInternalServerError)
	}
}

func (web Web) doAuth(w http.ResponseWriter, r *http.Request, sess session.Store) (*string, error) {
	ctx, sp := sl.OpenSpan(r.Context())
	defer sp.Close()

	state := r.FormValue("state")
	code := r.FormValue("code")
	stateValidate := sess.Get("state")

	ssoauth := authenticatorFromContext(r.Context())
	api := apiClientFromContext(r.Context())

	// I really need to read up on how this is useful, what I've read is that it's to help prevent man in the middle attacks?
	// But if they've intercepted the stream then they just return this... so I'm confused...
	// Good blog post about it's usefulness, I feel educated:
	// http://www.twobotechnologies.com/blog/2014/02/importance-of-state-in-oauth2.html
	if state != stateValidate {
		sp.Error("Invalid oauth state", zap.Any("expected_state", stateValidate), zap.String("actual_state", state))
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := ssoauth.TokenExchange(code)
	if err != nil {
		sp.Error("Code exchange failed", zap.Error(err))
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, err
	}

	tokenSource := ssoauth.TokenSource(token)
	// if err != nil {
	//	fmt.Printf("Token retrieve failed with '%s'\n", err)
	//	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	//	return nil, errors.New(fmt.Sprintf("Token retrieve failed with '%s'\n", err))
	// }

	verifyReponse, err := ssoauth.Verify(tokenSource)
	if err != nil {
		sp.Error("Error getting verify response", zap.Error(err))
		return nil, err
	}

	character, _, err := api.ESI.CharacterApi.GetCharactersCharacterId(r.Context(), int32(verifyReponse.CharacterID), nil)
	if err != nil {
		sp.Error("error getting character", zap.Error(err))
		return nil, err
	}

	corporation, _, err := api.ESI.CorporationApi.GetCorporationsCorporationId(r.Context(), character.CorporationId, nil)
	if err != nil {
		sp.Error("error getting corporation", zap.Error(err))
		return nil, err
	}

	var alliance esi.GetAlliancesAllianceIdOk
	if corporation.AllianceId != 0 {
		alliance, _, err = api.ESI.AllianceApi.GetAlliancesAllianceId(r.Context(), corporation.AllianceId, nil)
		if err != nil {
			sp.Error("error getting alliance", zap.Error(err))
			return nil, err
		}
	}

	// Auth internally, this is the source of the bot's auth code.
	// We know we'll have a corp and a character, we're not sure if the corp is in an alliance.
	request := &payloads.CreateRequest{
		Corporation: &payloads.Corporation{
			ID:     character.CorporationId,
			Name:   corporation.Name,
			Ticker: corporation.Ticker,
		},
		Character: &payloads.Character{
			ID:   verifyReponse.CharacterID,
			Name: character.Name,
		},
		Token: code,
		// TODO: When we implement custom scopes, send them over as well
		AuthScope: []string{"invalid"},
	}

	if corporation.AllianceId != 0 {
		request.Alliance = &payloads.Alliance{
			// TODO: Damn, why did I put int64 here?  At least I can upcast...
			ID:     corporation.AllianceId,
			Name:   alliance.Name,
			Ticker: alliance.Ticker,
		}
	}

	authCode, err := auth.Create(ctx, request, web.dependencies)

	if err != nil {
		sp.Error("Had an issue authing internally", zap.Error(err))
		return nil, err
	}

	return authCode, nil
}
