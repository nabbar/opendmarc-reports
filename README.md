# OpenDMARC Report generator

This tools import opendmarc history file into database and generate report from database
This tools run in multi thread process to limit time for generating report

This tools is based on OpenDmarc Perl script but without errors and with more efficient process

In add, this tools allow to send report in HTTP(s) / FTP(s) rua in add of Mail.

## Build the tools
It is never recommended to use directly built binary who's not included in offical os repository
In fact, if you find this tools in your distribution you can install it directly.
Otherwise, it is recommend to use the source to build it your self, in your secure environment.

For a best security, rebuilt this tools each week with this recommanded 3 steps process.

### 1 - Get the dependencies tools 
the build use the golang dep tools : https://golang.github.io/dep/
```shell
SRCTGZ=$(mktemp)
SRCURL=$(curl -s https://api.github.com/repos/golang/dep/releases/latest | jq '.tarball_url')
PKGDIR=$(echo "${GOPATH}/src/github.com/golang/dep")
[ -e ${PKGDIR} ] || mkdir -vp ${PKGDIR}
wget -O ${SRCTGZ} ${SRCURL} 
tar -xzf ${SRCTGZ} --strip 1 -C ${PKGDIR}
cd ${PKGDIR}
make install
```

### 2 - Get source in correct the correct package folder in go path:
We will download a tarball archive of the source
```shell
SRCTGZ=$(mktemp)
SRCURL=$(curl -s https://api.github.com/repos/nabbar/opendmarc-reports/releases/latest | jq '.tarball_url')
PKGDIR=$(echo "${GOPATH}/src/github.com/nabbar/opendmarc-reports")
[ -e ${PKGDIR} ] || mkdir -vp ${PKGDIR}
wget -O ${SRCTGZ} ${SRCURL}
tar -xzf ${SRCTGZ} --strip 1 -C ${PKGDIR}
cd ${PKGDIR}
```

### 3 - Regen Root Certificates
We will print all RootCA to include your own rootCA.
When running the script, we will use rootCA added of this list of CA
```shell
cd ${GOPATH}/src/github.com/nabbar/opendmarc-reports
./makeCertif
```

### 4 - Build the tools
```shell
cd ${GOPATH}/src/github.com/nabbar/opendmarc-reports
./genVersion
[ ! -e opendmarc-reports ] || mv -vf opendmarc-reports /usr/local/bin/opendmarc-reports
```

## Use the tools
Basicly running the tools without config file or params will show : 

```shell
allow to import history file into mysql DB,
generate report from this mysql DB normalized as OpenDMARC reports
and send them to MX server of each reports'domains'

Usage:
  opendmarc-reports [command]

Available Commands:
  config      Generate config file
  help        Help about any command
  import      Import dat history file
  report      Generate a report and send it

Flags:
  -c, --config string         config file (default is $HOME/.opendmarc.[yaml|json|toml])
  -d, --database string       Mysql Database params formatted as DSN string: <user>:<password>@protocol(<host>:<port>|<socket path>)/<database>[?[params[=value]]] (default "opendmarc:opendmarc@tcp(localhost:3306)/opendmarc")
  -y, --day                   Send report for yesterday's data (default true)
  -m, --domain strings        Force a report for named domain list (multiple flag allowed)
  -h, --help                  help for opendmarc-reports
  -i, --interval string       Report interval duration (default "24h")
  -e, --no-domain strings     Omit a report for named domain list (multiple flag allowed)
  -u, --no-update             Don't record report transmission
      --report-copy string    Report bcc email list (comma separated)
      --report-email string   Report email sender
      --report-org string     Report organisation sender
  -s, --smtp string           SMTP server params formatted as DSN string: <user>:<password>@tcp(<host|ip>:<port>)/[none|tls|starttls][?[serverName|skiptlsverify]=<value>] (default "postmaster@localdomain:opendmarc@tcp(localhost:25)/tls")
  -t, --test                  Don't send reports
  -z, --utc                   Operate in UTC
  -v, --verbose count         Enable verbose mode (multi allowed v, vv, vvv)
      --version               version for opendmarc-reports

Use "opendmarc-reports [command] --help" for more information about a command.

```

Note : the tables in the database will be create if they are not existing, but for security this tools will not create the database.
The right assign to the user must be in the database : CREATE TABLE, SELECT, INSERT, DELETE, UPDATE

### 1 - Generate a default config file 

Use the command "config" with args as you want to generate a config file.
The exact use is this : 

```shell
Generate a configuration file based on
giving existing config flag
override by passed flag in command line
and completed with default for non existing values
.

Usage:
  opendmarc-reports config <file path to be generated> [flags]

Examples:
config ~/.dmarc-reports.yml

Flags:
  -h, --help   help for config

Global Flags:
  -c, --config string         config file (default is $HOME/.opendmarc.[yaml|json|toml])
  -d, --database string       Mysql Database params formatted as DSN string: <user>:<password>@protocol(<host>:<port>|<socket path>)/<database>[?[params[=value]]] (default "opendmarc:opendmarc@tcp(localhost:3306)/opendmarc")
  -y, --day                   Send report for yesterday's data (default true)
  -m, --domain strings        Force a report for named domain list (multiple flag allowed)
  -i, --interval string       Report interval duration (default "24h")
  -e, --no-domain strings     Omit a report for named domain list (multiple flag allowed)
  -u, --no-update             Don't record report transmission
      --report-copy string    Report bcc email list (comma separated)
      --report-email string   Report email sender
      --report-org string     Report organisation sender
  -s, --smtp string           SMTP server params formatted as DSN string: <user>:<password>@tcp(<host|ip>:<port>)/[none|tls|starttls][?[serverName|skiptlsverify]=<value>] (default "postmaster@localdomain:opendmarc@tcp(localhost:25)/tls")
  -t, --test                  Don't send reports
  -z, --utc                   Operate in UTC
  -v, --verbose count         Enable verbose mode (multi allowed v, vv, vvv)

```

When generated you can use this config as default config by specify -c if not in the default path and default name.
The great interest of this file is to store your credentials but still allow you to override any config the current run without saving this overrides.
As the for example to use defautl config but for this run use UTC date/time, you can run a command like this :
```shell
opendmarc-reports <command> -c <path to no default config path> --utc <args...>
```

Once generated, you can modify the config file as you want or calling again the config command to overwrite your file with other default config

### 2 - Import history files
To import history file, the command is "import".
By default this tools will looking for job id in database and if find a same jobid, It will update it, otherwise it will insert it.
The import command will run a multi thread of each path given. The path could be folder, file or pattern.
the help of this command will show : 

```shell
Import OpenDMARC history file
into mysql database. If not exist create
the record else update it.

Usage:
  opendmarc-reports import <dat file pattern> [<dat file pattern>, ...] [flags]

Examples:
import /var/tmp/dmarc.dat /var/tmp/opendmarc.*

Flags:
  -h, --help   help for import

Global Flags:
  -c, --config string         config file (default is $HOME/.opendmarc.[yaml|json|toml])
  -d, --database string       Mysql Database params formatted as DSN string: <user>:<password>@protocol(<host>:<port>|<socket path>)/<database>[?[params[=value]]] (default "opendmarc:opendmarc@tcp(localhost:3306)/opendmarc")
  -y, --day                   Send report for yesterday's data (default true)
  -m, --domain strings        Force a report for named domain list (multiple flag allowed)
  -i, --interval string       Report interval duration (default "24h")
  -e, --no-domain strings     Omit a report for named domain list (multiple flag allowed)
  -u, --no-update             Don't record report transmission
      --report-copy string    Report bcc email list (comma separated)
      --report-email string   Report email sender
      --report-org string     Report organisation sender
  -s, --smtp string           SMTP server params formatted as DSN string: <user>:<password>@tcp(<host|ip>:<port>)/[none|tls|starttls][?[serverName|skiptlsverify]=<value>] (default "postmaster@localdomain:opendmarc@tcp(localhost:25)/tls")
  -t, --test                  Don't send reports
  -z, --utc                   Operate in UTC
  -v, --verbose count         Enable verbose mode (multi allowed v, vv, vvv)

```

This command will not modify any file !

### 3 - Generate and Send report
To send the report to each rua of db store job, use the "report" command.
The process will make a thread for each rua domain * rua request * rua protocol destination.
This feature is interessting to having a running time like the more long send report to one destination.

In this case, this tools will open multiple connection to SMTP server, HTTP(s) destination, FPT(s) destination.
In your SMTP server, if you have a DKIM signature process the generated mail will use it.

the help for the report command is :

```shell
Load OpenDMARC history data from mysql database,
generate report for selected domains or all domains,
and sent it by mail through SMTP server.

Usage:
  opendmarc-reports report [flags]

Examples:
report

Flags:
  -h, --help   help for report

Global Flags:
  -c, --config string         config file (default is $HOME/.opendmarc.[yaml|json|toml])
  -d, --database string       Mysql Database params formatted as DSN string: <user>:<password>@protocol(<host>:<port>|<socket path>)/<database>[?[params[=value]]] (default "opendmarc:opendmarc@tcp(localhost:3306)/opendmarc")
  -y, --day                   Send report for yesterday's data (default true)
  -m, --domain strings        Force a report for named domain list (multiple flag allowed)
  -i, --interval string       Report interval duration (default "24h")
  -e, --no-domain strings     Omit a report for named domain list (multiple flag allowed)
  -u, --no-update             Don't record report transmission
      --report-copy string    Report bcc email list (comma separated)
      --report-email string   Report email sender
      --report-org string     Report organisation sender
  -s, --smtp string           SMTP server params formatted as DSN string: <user>:<password>@tcp(<host|ip>:<port>)/[none|tls|starttls][?[serverName|skiptlsverify]=<value>] (default "postmaster@localdomain:opendmarc@tcp(localhost:25)/tls")
  -t, --test                  Don't send reports
  -z, --utc                   Operate in UTC
  -v, --verbose count         Enable verbose mode (multi allowed v, vv, vvv)
```

## Contribute

The day have only 24h and so I will thanks you a lot if you want contribute.

