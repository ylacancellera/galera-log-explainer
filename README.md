[![Go Report Card](https://goreportcard.com/badge/github.com/ylacancellera/galera-log-explainer)](https://goreportcard.com/report/github.com/ylacancellera/galera-log-explainer) 

# galera-log-explainer

Filter, aggregate and summarize multiple galera logs together.


## Features

* List events in chronological order from any number of nodes
* List key points of information from logs (sst, view changes, general errors, maintenance operations)
* Translate advanced Galera information to a easily readable counterpart
* Filter on dates with --since, --until
* Filter on type of events
* Aggregates rotated logs together, even when there are logs from multiple nodes


Get the latest cluster changes on a local server
```sh
galera-log-explainer list --all --since 2023-01-05T03:24:26.000000Z /var/log/mysql/*.log
```

Find out information about nodes, using any type of info
```sh
galera-log-explainer whois 'galera-node2' mysql.log 
{
	"input": "218469b2",
	"IPs": [
		"172.17.0.3"
	],
	"nodeNames": [
		"galera-node2"
	],
	"hostname": "",
	"nodeUUIDs:": [
		"218469b2",
		"259b78a0",
		"fa81213d",
	]
}
```

You can find information from UUIDs, IPs, node names
```
galera-log-explainer whois '172.17.0.3' mysql.log 

galera-log-explainer whois 'galera-node2' mysql.log 
```

Automatically translate every information (IP, UUID) from a log
```
galera-log-explainer sed some/log.log another/one.log to_translate.log < to_translate.log  | less

cat to_translate.log | galera-log-explainer sed some/log.log another/one.log to_translate.log | less
```
Or get the raw `sed` command to do it yourself
```
galera-log-explainer sed some/log.log another/one.log to_translate.log
```

Usage:

	$ galera-log-explainer --help
	Usage: galera-log-explainer <command>

	An utility to transform Galera logs in a readable version
	
	Flags:
	  -h, --help               Show context-sensitive help.
	      --no-color
	      --since=SINCE        Only list events after this date, you can copy-paste a date from mysql
	                           error log
	      --until=UNTIL        Only list events before this date, you can copy-paste a date from mysql
	                           error log
	      --verbosity=1        0: Info, 1: Detailed, 2: DebugMySQL (every mysql info the tool used), 3:
	                           Debug (internal tool debug)
	      --pxc-operator       Analyze logs from Percona PXC operator. Off by default because it
	                           negatively impacts performance for non-k8s setups
	      --exclude-regexes=EXCLUDE-REGEXES,...
	                           Remove regexes from analysis. List regexes using 'galera-log-explainer
	                           regex-list'
	      --grep-cmd="grep"    'grep' command path. Could need to be set to 'ggrep' for darwin systems
	      --grep-args="-P"     'grep' arguments. perl regexp (-P) is necessary. -o will break the tool
	
	Commands:
	  list <paths> ...
	
	  whois <search> <paths> ...
	
	  sed <paths> ...
	
	  ctx <paths> ...
	
	  regex-list
	
	Run "galera-log-explainer <command> --help" for more information on a command.


## Compatibility

* Percona XtraDB Cluster: 5.5 to 8.0
* MariaDB Galera Cluster: 10.0 to 10.6
* Galera logs from K8s pods
