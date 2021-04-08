package discord

import (
	"fmt"

	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

//var matchDiscordError = regexp.MustCompile(`^The role '.*' already exists$`)

type Discord struct {
	logger *zap.SugaredLogger
}

func New(logger *zap.SugaredLogger) *Discord {
	return &Discord{logger: logger}
}

func (d Discord) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		return nil
	}

	fmt.Println(m.Body)

	// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
	return nil
}

//func (d Discord) AddDiscordRole(name string, logger *zap.SugaredLogger, s *discordgo.Session) error {
//	// TODO: push to queue
//	_, err := h.clients.discord.CreateRole(ctx, &discord.CreateRoleRequest{Name: name})
//	if err != nil {
//		if matchDiscordError.MatchString(err.Error()) {
//			// The role list was cached most likely so we'll pretend we didn't try
//			// to create it just now. -brian
//		} else {
//			msg := fmt.Sprintf("addDiscordRole '%s': %s", name, err.Error())
//			logger.Error(msg)
//			return err
//		}
//	}
//
//	return nil
//}
//
//func CreateRole() string {
//	err := dgh.roleMap.UpdateRoles()
//	if err != nil {
//		dgh.Logger.Error(err.Error())
//		return err
//	}
//
//	allRoles := dgh.roleMap.GetRoles()
//
//	for key := range allRoles {
//		if allRoles[key].Name == request.Name {
//			dgh.Logger.Sugar().Errorf("The role '%s' already exists", allRoles[key].Name)
//			return fmt.Errorf("The role '%s' already exists", allRoles[key].Name)
//		}
//	}
//
//	role, err := dgh.client.CreateRole(dgh.discordServerId)
//	if err != nil {
//		dgh.Logger.Error(err.Error())
//		return err
//	}
//
//	editedRole, err := dgh.client.EditRole(dgh.discordServerId, role.ID, request.Name, int(request.Color), int64(request.Permissions), request.Hoist, request.Mentionable)
//	if err != nil {
//		deleteErr := dgh.client.DeleteRole(dgh.discordServerId, role.ID)
//		if deleteErr != nil {
//			dgh.Logger.Sugar().Errorf("edit failure (%s), delete failure (%s)", err.Error(), deleteErr.Error())
//			return errors.New(fmt.Sprintf("edit failure (%s), delete failure (%s)", err.Error(), deleteErr.Error()))
//		}
//
//		dgh.Logger.Error(err.Error())
//		return err
//	}
//
//	//Now validate the edited role
//	if !validateRole(request, editedRole) {
//		err = dgh.client.DeleteRole(dgh.discordServerId, role.ID)
//		if err != nil {
//			dgh.Logger.Sugar().Errorf("attempted to delete role due to invalid response but received error (%s)", err.Error())
//			return errors.New(fmt.Sprintf("attempted to delete role due to invalid response but received error (%s)", err.Error()))
//		}
//
//		dgh.Logger.Error("role create failed due to invalid response from discord")
//		return errors.New("role create failed due to invalid response from discord")
//	}
//
//	response.RoleId = editedRole.ID
//
//	// Reset cache as we've made changes to discord that need to be picked up next run
//	dgh.lastRoleCall = dgh.lastRoleCall.AddDate(0, 0, -1)
//	return nil
//}
