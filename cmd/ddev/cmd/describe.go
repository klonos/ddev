package cmd

import (
	"fmt"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:   "describe [projectname]",
	Short: "Get a detailed description of a running ddev project.",
	Long: `Get a detailed description of a running ddev project. Describe provides basic
information about a ddev project, including its name, location, url, and status.
It also provides details for MySQL connections, and connection information for
additional services like MailHog and phpMyAdmin. You can run 'ddev describe' from
a project directory to describe that project, or you can specify a project to describe by
running 'ddev describe <projectname>.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev describe' or 'ddev describe [projectname]'")
		}

		projects, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Failed to describe project(s): %v", err)
		}
		project := projects[0]

		if err := ddevapp.CheckForMissingProjectFiles(project); err != nil {
			util.Failed("Failed to describe %s: %v", project.Name, err)
		}

		desc, err := project.Describe()
		if err != nil {
			util.Failed("Failed to describe project %s: %v", project.Name, err)
		}

		renderedDesc, err := renderAppDescribe(desc)
		util.CheckErr(err) // We shouldn't ever end up with an unrenderable desc.
		output.UserOut.WithField("raw", desc).Print(renderedDesc)
	},
}

// renderAppDescribe takes the map describing the app and renders it for plain-text output
func renderAppDescribe(desc map[string]interface{}) (string, error) {

	maxWidth := uint(200)
	var output string

	status := desc["status"]

	appTable := ddevapp.CreateAppTable()
	ddevapp.RenderAppRow(appTable, desc)
	output = fmt.Sprint(appTable)

	// Only show extended status for running sites.
	if status == ddevapp.SiteRunning {
		output = output + "\n\nProject Information\n-----------------\n"
		siteInfo := uitable.New()
		siteInfo.AddRow("PHP version:", desc["php_version"])

		siteInfo.AddRow("URLs:", strings.Join(desc["urls"].([]string), ", "))
		output = output + fmt.Sprint(siteInfo)
		dockerIP, err := dockerutil.GetDockerIP()
		if err != nil {
			return "", err
		}

		output = output + "\n\nMySQL Credentials\n-----------------\n"
		dbTable := uitable.New()

		dbinfo := desc["dbinfo"].(map[string]interface{})

		if _, ok := dbinfo["username"].(string); ok {
			dbTable.MaxColWidth = maxWidth
			dbTable.AddRow("Username:", dbinfo["username"])
			dbTable.AddRow("Password:", dbinfo["password"])
			dbTable.AddRow("Database name:", dbinfo["dbname"])
			dbTable.AddRow("Host:", dbinfo["host"])
			dbTable.AddRow("Port:", dbinfo["port"])
			dbTable.AddRow("MariaDB version", dbinfo["mariadb_version"])
			output = output + fmt.Sprint(dbTable)
			output = output + fmt.Sprintf("\nTo connect to mysql from your host machine, use port %d on %s.\nFor example: mysql --host=%s --port=%d --user=db --password=db --database=db", dbinfo["published_port"], dockerIP, dockerIP, dbinfo["published_port"])
		}
		output = output + "\n\nOther Services\n--------------\n"
		other := uitable.New()
		other.AddRow("MailHog:", desc["mailhog_url"])
		if _, ok := desc["phpmyadmin_url"]; ok {
			other.AddRow("phpMyAdmin:", desc["phpmyadmin_url"])
		}
		output = output + fmt.Sprint(other)

		output = output + "\n" + ddevapp.RenderRouterStatus() + "\t" + ddevapp.RenderSSHAuthStatus()
	}

	return output, nil
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
