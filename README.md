# galera-log-explainer

Filter, aggregate and summarize multiple galera logs together.


## Features

* List events in chronological order from any number of nodes
* List key points of information from logs (sst, view changes, general errors, maintenance operations)
* Translate advanced Galera information to a easily readable counterpart
* Filter on dates with --since, --until
* Filter on type of events 


Get the latest events on a local server
```sh
galera-log-explainer list --all --since 2023-03-24T10:24:00.000000Z /var/log/mysql/*.log
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

Automatically translate every information (IP, UUID) to the node name in a file
```
galera-log-explainer sed some/logs/to/analyze.log another/one.log mysql_log_to_translate.log < mysql_log_to_translate.log  | less

cat mysql_log_to_translate.log | galera-log-explainer sed some/logs/to/analyze.log another/one.log mysql_log_to_translate.log | less
```


Usage:
	$ galera-log-explainer --help
	Usage: galera-log-explainer <command>
	
	An utility to transform Galera logs in a readable version
	
	Flags:
	  -h, --help           Show context-sensitive help.
	      --no-color
	      --since=SINCE    Only list events after this date, you can copy-paste a date from mysql error log
	      --until=UNTIL    Only list events before this date, you can copy-paste a date from mysql error log
	      --verbosity=1    0: Info, 1: Detailed, 2: DebugMySQL (every mysql info the tool used), 3: Debug
	                       (internal tool debug)
	
	Commands:
	  list <paths> ...
	
	  whois <search> <paths> ...
	
	  sed <paths> ...
	
	  summary <paths> ...
	
	Run "galera-log-explainer <command> --help" for more information on a command.

