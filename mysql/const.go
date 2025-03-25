package mysql

const (
	PARAM_UNSIGNED = 128
)

const (
	COM_SLEEP               byte = 0
	COM_QUIT                byte = 1
	COM_INIT_DB             byte = 2
	COM_QUERY               byte = 3
	COM_FIELD_LIST          byte = 4
	COM_CREATE_DB           byte = 5
	COM_DROP_DB             byte = 6
	COM_REFRESH             byte = 7
	COM_SHUTDOWN            byte = 8
	COM_STATISTICS          byte = 9
	COM_PROCESS_INFO        byte = 10
	COM_CONNECT             byte = 11
	COM_PROCESS_KILL        byte = 12
	COM_DEBUG               byte = 13
	COM_PING                byte = 14
	COM_TIME                byte = 15
	COM_DELAYED_INSERT      byte = 16
	COM_CHANGE_USER         byte = 17
	COM_BINLOG_DUMP         byte = 18
	COM_TABLE_DUMP          byte = 19
	COM_CONNECT_OUT         byte = 20
	COM_REGISTER_SLAVE      byte = 21
	COM_STMT_PREPARE        byte = 22
	COM_STMT_EXECUTE        byte = 23
	COM_STMT_SEND_LONG_DATA byte = 24
	COM_STMT_CLOSE          byte = 25
	COM_STMT_RESET          byte = 26
	COM_SET_OPTION          byte = 27
	COM_STMT_FETCH          byte = 28
	COM_DAEMON              byte = 29
	COM_BINLOG_DUMP_GTID    byte = 30
	COM_RESET_CONNECTION    byte = 31
)

const (
	MYSQL_TYPE_DECIMAL   byte = 0
	MYSQL_TYPE_TINY      byte = 1
	MYSQL_TYPE_SHORT     byte = 2
	MYSQL_TYPE_LONG      byte = 3
	MYSQL_TYPE_FLOAT     byte = 4
	MYSQL_TYPE_DOUBLE    byte = 5
	MYSQL_TYPE_NULL      byte = 6
	MYSQL_TYPE_TIMESTAMP byte = 7
	MYSQL_TYPE_LONGLONG  byte = 8
	MYSQL_TYPE_INT24     byte = 9
	MYSQL_TYPE_DATE      byte = 10
	MYSQL_TYPE_TIME      byte = 11
	MYSQL_TYPE_DATETIME  byte = 12
	MYSQL_TYPE_YEAR      byte = 13
	MYSQL_TYPE_NEWDATE   byte = 14
	MYSQL_TYPE_VARCHAR   byte = 15
	MYSQL_TYPE_BIT       byte = 16
)

const (
	MYSQL_TYPE_JSON byte = iota + 0xf5
	MYSQL_TYPE_NEWDECIMAL
	MYSQL_TYPE_ENUM
	MYSQL_TYPE_SET
	MYSQL_TYPE_TINY_BLOB
	MYSQL_TYPE_MEDIUM_BLOB
	MYSQL_TYPE_LONG_BLOB
	MYSQL_TYPE_BLOB
	MYSQL_TYPE_VAR_STRING
	MYSQL_TYPE_STRING
	MYSQL_TYPE_GEOMETRY
)
