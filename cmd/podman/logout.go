package main

import (
	"fmt"

	"github.com/containers/image/docker"
	"github.com/containers/image/pkg/docker/config"
	"github.com/containers/libpod/cmd/podman/cliconfig"
	"github.com/containers/libpod/libpod/image"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	logoutCommand     cliconfig.LogoutValues
	logoutDescription = "Remove the cached username and password for the registry."
	_logoutCommand    = &cobra.Command{
		Use:   "logout [flags] REGISTRY",
		Short: "Logout of a container registry",
		Long:  logoutDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			logoutCommand.InputArgs = args
			logoutCommand.GlobalFlags = MainGlobalOpts
			logoutCommand.Remote = remoteclient
			return logoutCmd(&logoutCommand)
		},
		Example: `podman logout docker.io
  podman logout --authfile authdir/myauths.json docker.io
  podman logout --all`,
	}
)

func init() {
	logoutCommand.Command = _logoutCommand
	logoutCommand.SetHelpTemplate(HelpTemplate())
	logoutCommand.SetUsageTemplate(UsageTemplate())
	flags := logoutCommand.Flags()
	flags.BoolVarP(&logoutCommand.All, "all", "a", false, "Remove the cached credentials for all registries in the auth file")
	flags.StringVar(&logoutCommand.Authfile, "authfile", "", "Path of the authentication file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json. Use REGISTRY_AUTH_FILE environment variable to override")

}

// logoutCmd uses the authentication package to remove the authenticated of a registry
// stored in the auth.json file
func logoutCmd(c *cliconfig.LogoutValues) error {
	args := c.InputArgs
	if len(args) > 1 {
		return errors.Errorf("too many arguments, logout takes at most 1 argument")
	}
	if len(args) == 0 && !c.All {
		return errors.Errorf("registry must be given")
	}
	var server string
	if len(args) == 1 {
		server = scrubServer(args[0])
	}
	authfile := getAuthFile(c.Authfile)

	sc := image.GetSystemContext("", authfile, false)

	if c.All {
		if err := config.RemoveAllAuthentication(sc); err != nil {
			return err
		}
		fmt.Println("Removed login credentials for all registries")
		return nil
	}

	err := config.RemoveAuthentication(sc, server)
	switch errors.Cause(err) {
	case nil:
		fmt.Printf("Removed login credentials for %s\n", server)
		return nil
	case config.ErrNotLoggedIn:
		// username of user logged in to server (if one exists)
		userFromAuthFile, passFromAuthFile, err := config.GetAuthentication(sc, server)
		if err != nil {
			return errors.Wrapf(err, "error reading auth file")
		}
		islogin := docker.CheckAuth(getContext(), sc, userFromAuthFile, passFromAuthFile, server)
		if userFromAuthFile != "" && passFromAuthFile != "" && islogin == nil {
			fmt.Printf("Not logged into %s with podman. Existing credentials were established via docker login. Please use docker logout instead.\n", server)
			return nil
		}
		fmt.Printf("Not logged into %s\n", server)
		return nil
	default:
		return errors.Wrapf(err, "error logging out of %q", server)
	}
}
