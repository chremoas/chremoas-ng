package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/astaxie/beego/session"
	"github.com/chremoas/chremoas-ng/internal/auth"
	abaeve_auth "github.com/chremoas/chremoas-ng/internal/auth-srv/proto"
	"github.com/dimfeld/httptreemux"
	"github.com/gregjones/httpcache"
	"github.com/spf13/viper"
)

const (
	name = "auth"
)

var (
	globalSessions *session.Manager
	apiClient      *goesi.APIClient
	authenticator  *goesi.SSOAuthenticator
	templates      = template.New("")
)

type ResultModel struct {
	Title      string
	Auth       string
	DiscordUrl string
	Name       string
}

type Web string

func New() (*Web, error) {
	var web Web

	//Setup our required globals.
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

	//Initialize my templates
	for _, path := range AssetNames() {
		bytes, err := Asset(path)
		if err != nil {
			log.Printf("Unable to parse: path=%s, err=%s", path, err)
			return nil, fmt.Errorf("Unable to parse: path=%s, err=%s", path, err)
		}
		templates.New(path).Parse(string(bytes))
	}

	return &web, nil
}

func (w Web) Auth() *httptreemux.ContextMux {
	mux := httptreemux.NewContextMux()

	mux.Handle(http.MethodGet, "/ready", readiness)
	mux.Handle(http.MethodGet, "/static/*path", serveFiles)
	mux.Handle(http.MethodGet, "/", middleware(handleIndex))
	mux.Handle(http.MethodGet, "/login", middleware(handleEveLogin))
	mux.Handle(http.MethodGet, viper.GetString("oauth.callBackUrl"), middleware(handleEveCallback))

	return mux
}

func serveFiles(_ http.ResponseWriter, _ *http.Request) {
	http.StripPrefix("/static/",
		http.FileServer(assetFS()))
}

func readiness(w http.ResponseWriter, _ *http.Request) {
	status := struct {
		Status string
	}{
		Status: "OK",
	}
	json.NewEncoder(w).Encode(status)
}

func handleIndex(w http.ResponseWriter, _ *http.Request) {
	templates.ExecuteTemplate(w, "templates/index.html", nil)
}

func handleEveLogin(w http.ResponseWriter, r *http.Request) {
	// Get the users session
	sess, _ := globalSessions.SessionStart(w, r)

	// Get the authenticator from the request context
	ssoauth := authenticatorFromContext(r.Context())

	// Generate a random 16 byte state.
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	// Save the state to the session to validate with the response.
	sess.Set("state", state)

	// Build the authorize URL
	//TODO: This is where we'd set extra needed scopes
	redirectUrl := ssoauth.AuthorizeURL(state, true, nil)

	// Redirect the user to CCP SSO
	http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
}

func handleEveCallback(w http.ResponseWriter, r *http.Request) {
	sess, _ := globalSessions.SessionStart(w, r)
	if sess == nil {
		fmt.Print("No session, redirecting to /\n")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	internalAuthCode, err := doAuth(w, r, sess)
	if err != nil {
		//TODO: Make another template for errors specifically for this endpoint
		fmt.Printf("Received an error from doAuth: (%s)\n", err)
		return
	}

	templates.ExecuteTemplate(w, "templates/authd.html",
		&ResultModel{
			Title:      "Authd Up",
			Auth:       *internalAuthCode,
			DiscordUrl: viper.GetString("discord.inviteUrl"),
			Name:       name,
		},
	)
}

func doAuth(w http.ResponseWriter, r *http.Request, sess session.Store) (*string, error) {
	state := r.FormValue("state")
	code := r.FormValue("code")
	stateValidate := sess.Get("state")

	ssoauth := authenticatorFromContext(r.Context())
	api := apiClientFromContext(r.Context())

	//I really need to read up on how this is useful, what I've read is that it's to help prevent man in the middle attacks?
	//But if they've intercepted the stream then they just return this... so I'm confused...
	//Good blog post about it's usefulness, I feel educated:
	//http://www.twobotechnologies.com/blog/2014/02/importance-of-state-in-oauth2.html
	if state != stateValidate {
		fmt.Printf("Invalid oauth state, expected '%s', got '%s'\n", stateValidate, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, errors.New(fmt.Sprintf("Invalid oauth state, expected '%s', got '%s'\n", stateValidate, state))
	}

	token, err := ssoauth.TokenExchange(code)
	if err != nil {
		fmt.Printf("Code exchange failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, errors.New(fmt.Sprintf("Code exchange failed with '%s'\n", err))
	}

	tokenSource := ssoauth.TokenSource(token)
	//if err != nil {
	//	fmt.Printf("Token retrieve failed with '%s'\n", err)
	//	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	//	return nil, errors.New(fmt.Sprintf("Token retrieve failed with '%s'\n", err))
	//}

	verifyReponse, err := ssoauth.Verify(tokenSource)
	if err != nil {
		fmt.Printf("Had some kind of error getting the verify response '%s'\n", err)
	}

	character, _, err := api.ESI.CharacterApi.GetCharactersCharacterId(context.Background(), int32(verifyReponse.CharacterID), nil)
	if err != nil {
		fmt.Printf("Had some kind of error getting the character '%s'\n", err)
	}

	corporation, _, err := api.ESI.CorporationApi.GetCorporationsCorporationId(context.Background(), character.CorporationId, nil)
	if err != nil {
		fmt.Printf("Had some kind of error getting the corporation '%s'\n", err)
	}

	var alliance esi.GetAlliancesAllianceIdOk
	if corporation.AllianceId != 0 {
		alliance, _, err = api.ESI.AllianceApi.GetAlliancesAllianceId(context.Background(), corporation.AllianceId, nil)
		if err != nil {
			fmt.Printf("Had some kind of error getting the alliance '%s'\n", err)
		}
	}

	//Auth internally, this is the source of the bot's auth code.
	//We know we'll have a corp and a character, we're not sure if the corp is in an alliance.
	request := &abaeve_auth.AuthCreateRequest{
		Corporation: &abaeve_auth.Corporation{
			Id:     int64(character.CorporationId),
			Name:   corporation.Name,
			Ticker: corporation.Ticker,
		},
		Character: &abaeve_auth.Character{
			Id:   int64(verifyReponse.CharacterID),
			Name: character.Name,
		},
		Token: code,
		//TODO: When we implement custom scopes, send them over as well
		AuthScope: []string{"invalid"},
	}

	if corporation.AllianceId != 0 {
		request.Alliance = &abaeve_auth.Alliance{
			//TODO: Damn, why did I put int64 here?  At least I can upcast...
			Id:     int64(corporation.AllianceId),
			Name:   alliance.Name,
			Ticker: alliance.Ticker,
		}
	}

	response, err := auth.Create(context.Background(), request)

	fmt.Printf("%v\n", response)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Had an issue authing internally: (%s)", err))
	}

	if !response.Success {
		return nil, errors.New("received a non-specific error from the internal auth server, please contact your administrator")
	}

	return &response.AuthenticationCode, nil
}
