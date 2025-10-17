package config

import (
	"fmt"
	"os"

	"github.com/Netflix/go-env"
	"github.com/joho/godotenv"
)

type Configuration struct {
	Database struct {
		File string `env:"DB_FILE,default=freeipa-runner.db"`
	}

	LDAP struct {
		Address      string `env:"LDAP_ADDRESS,required=true"`
		DomainSLD    string `env:"LDAP_DOMAIN_SLD,required=true"`
		DomainTLD    string `env:"LDAP_DOMAIN_TLD,required=true"`
		AccountsCN   string `env:"LDAP_ACCOUNTS_CN,default=accounts"`
		UsersCN      string `env:"LDAP_USERS_CN,default=users"`
		GroupsCN     string `env:"LDAP_GROUPS_CN,default=groups"`
		HostsCN      string `env:"LDAP_HOSTS_CN,default=hosts"`
		HostGroupsCN string `env:"LDAP_HOST_GROUPS_CN,default=hostgroups"`
		ServicesCN   string `env:"LDAP_SERVICES_CN,default=services"`
		BindUsername string `env:"LDAP_BIND_USERNAME,required=true"`
		BindPassword string `env:"LDAP_BIND_PASSWORD,required=true"`
	}

	SSH struct {
		User              string `env:"SSH_USER,default=admin"`
		UseKerberos       bool   `env:"SSH_USE_KERBEROS,default=true"`
		PrivateKeyPath    string `env:"SSH_KEY_PATH,default="`
		KnownHostsPath    string `env:"SSH_KNOWN_HOSTS,default=~/.ssh/known_hosts"`
		Concurrency       int    `env:"SSH_CONCURRENCY,default=10"`
		Sudo              bool   `env:"SSH_SUDO,default=true"`
		TimeoutSeconds    int    `env:"SSH_TIMEOUT,default=900"`
		SystemdUnitPrefix string `env:"SSH_SYSTEMD_UNIT_PREFIX,default=freeipa-task"`
	}
}

var Config Configuration

// Try to initialize the environment variables from a .env in the directory the program is run from.
// If the .env file is not present, we will create a sample .env file based on the Configuration struct.
// You can then use config.Config globally
func InitEnv(path string) error {
	if _, err := os.Stat(path); err != nil {
		if e := GenerateSampleEnvFile(path); e != nil {
			return e
		}

		return fmt.Errorf("no .env file found, created a sample .env file. Please fill in the required values and try again")
	}

	if err := godotenv.Load(path); err != nil {
		return err
	}

	_, err := env.UnmarshalFromEnviron(&Config)
	if err != nil {
		return err
	}

	return nil
}
