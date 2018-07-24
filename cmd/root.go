package cmd

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/nabbar/opendmarc-reports/config"
	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/version"
	"github.com/spf13/cobra"
	"github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

/*
Copyright 2018 Nicolas JUHEL

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

var (
	cfgFile string

	flgVerbose  int
	flgTest     bool
	flgNoUpd    bool
	flgInterval string
	flgUTC      bool
	flgDay      bool
	flgDomain   []string
	flgNoDomain []string

	flgDBDSN string
	flgSMTP  string

	flgReportEmail string
	flgReportOrg   string
	flgReportCopy  string

	flgDATPath []string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     version.Package,
	Version: fmt.Sprintf("%s\n%s\n%s\n", version.GetAppId(), version.GetInfo(), version.GetAuthor()),
	Short:   "Manage OpenDMARC report and history",
	Long: `allow to import history file into mysql DB,
generate report from this mysql DB normalized as OpenDMARC reports
and send them to MX server of each reports'domains'`,
	TraverseChildren: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		config.GetConfig()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	viper.SetEnvPrefix(version.GetPrefix())
	cobra.OnInitialize(initConfig)
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.opendmarc.[yaml|json|toml])")

	rootCmd.PersistentFlags().CountVarP(&flgVerbose, "verbose", "v", "Enable verbose mode (multi allowed v, vv, vvv)")
	rootCmd.PersistentFlags().BoolVarP(&flgTest, "test", "t", false, "Don't send reports")
	rootCmd.PersistentFlags().BoolVarP(&flgNoUpd, "no-update", "u", false, "Don't record report transmission")
	rootCmd.PersistentFlags().StringVarP(&flgInterval, "interval", "i", config.DEFAULT_INTERVAL, "Report interval duration")
	rootCmd.PersistentFlags().BoolVarP(&flgUTC, "utc", "z", false, "Operate in UTC")
	rootCmd.PersistentFlags().BoolVarP(&flgDay, "day", "y", true, "Send report for yesterday's data")
	rootCmd.PersistentFlags().StringSliceVarP(&flgDomain, "domain", "m", make([]string, 0), "Force a report for named domain list (multiple flag allowed)")
	rootCmd.PersistentFlags().StringSliceVarP(&flgNoDomain, "no-domain", "e", make([]string, 0), "Omit a report for named domain list (multiple flag allowed)")

	rootCmd.PersistentFlags().StringVarP(&flgDBDSN, "database", "d", config.GetDefaultDSN(), "Mysql Database params formatted as DSN string: <user>:<password>@protocol(<host>:<port>|<socket path>)/<database>[?[params[=value]]]")
	rootCmd.PersistentFlags().StringVarP(&flgSMTP, "smtp", "s", config.GetDefaultSmtp(), "SMTP server params formatted as DSN string: <user>:<password>@tcp(<host|ip>:<port>)/[none|tls|starttls][?[serverName|skiptlsverify]=<value>]")

	rootCmd.PersistentFlags().StringVar(&flgReportEmail, "report-email", "", "Report email sender")
	rootCmd.PersistentFlags().StringVar(&flgReportOrg, "report-org", "", "Report organisation sender")
	rootCmd.PersistentFlags().StringVar(&flgReportCopy, "report-copy", "", "Report bcc email list (comma separated)")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("testing", rootCmd.PersistentFlags().Lookup("test"))
	viper.BindPFlag("noUpdate", rootCmd.PersistentFlags().Lookup("no-update"))
	viper.BindPFlag("yesterday", rootCmd.PersistentFlags().Lookup("day"))

	viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))
	viper.BindPFlag("utc", rootCmd.PersistentFlags().Lookup("utc"))
	viper.BindPFlag("database", rootCmd.PersistentFlags().Lookup("database"))
	viper.BindPFlag("smtp", rootCmd.PersistentFlags().Lookup("smtp"))

	viper.BindPFlag("report.email", rootCmd.PersistentFlags().Lookup("report-email"))
	viper.BindPFlag("report.org", rootCmd.PersistentFlags().Lookup("report-org"))
	viper.BindPFlag("report.copy", rootCmd.PersistentFlags().Lookup("report-copy"))

	viper.BindPFlag("domain.only", rootCmd.PersistentFlags().Lookup("domain"))
	viper.BindPFlag("domain.exclude", rootCmd.PersistentFlags().Lookup("no-domain"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	switch flgVerbose {
	case 0:
		SetLevel(ErrorLevel.String())
		jwalterweatherman.SetStdoutThreshold(jwalterweatherman.LevelError)
	case 1:
		SetLevel(WarnLevel.String())
		jwalterweatherman.SetStdoutThreshold(jwalterweatherman.LevelWarn)
	case 2:
		SetLevel(InfoLevel.String())
		jwalterweatherman.SetStdoutThreshold(jwalterweatherman.LevelInfo)
	default:
		SetLevel(DebugLevel.String())
		jwalterweatherman.SetStdoutThreshold(jwalterweatherman.LevelDebug)
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".opendmarc" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".opendmarc")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
