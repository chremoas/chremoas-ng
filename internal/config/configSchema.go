package config

type Configuration struct {
	Namespace   string
	Database    struct {
		Driver         string
		Protocol       string
		Host           string
		Port           uint
		AuthDatabase   string
		RoleDatabase   string
		Username       string
		Password       string
		Options        string
		MaxConnections int `yaml:"maxConnections"`
	}
	Bot struct {
		Token           string   `yaml:"botToken"`
		DiscordServerId string   `yaml:"discordServerId"`
		Role            string   `yaml:"botRole"`
		IgnoredRoles    []string `yaml:"ignoredRoles"`
	}
	OAuth struct {
		ClientId         string `yaml:"clientId"`
		ClientSecret     string `yaml:"clientSecret"`
		CallBackProtocol string `yaml:"callBackProtocol"`
		CallBackHost     string `yaml:"callBackHost"`
		CallBackUrl      string `yaml:"callBackUrl"`
	} `yaml:"oauth"`
	Net struct {
		ListenHost string `yaml:"listenHost"`
		ListenPort int    `yaml:"listenPort"`
	}
	Discord struct {
		InviteUrl string `yaml:"inviteUrl"`
	} `yaml:"discord"`
	Registry struct {
		Hostname         string `yaml:"hostname"`
		Port             int    `yaml:"port"`
		RegisterTTL      int    `yaml:"registerTtl"`
		RegisterInterval int    `yaml:"registerInterval"`
	} `yaml:"registry"`
	Inputs []string `yaml:"inputs"`
	Chat   struct {
		Slack struct {
			Debug bool   `yaml:"debug"`
			Token string `yaml:"token"`
		} `yaml:"slack"`
		Discord struct {
			Token     string   `yaml:"token"`
			WhiteList []string `yaml:"whiteList"`
			Prefix    string   `yaml:"prefix"`
		} `yaml:"discord"`
	} `yaml:"chat"`
	Extensions map[interface{}]interface{} `yaml:"extensions"`
}
