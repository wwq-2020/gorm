CREATE TABLE user (
	id bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
	name varchar(100)  NOT NULL DEFAULT '' COMMENT '名字',
	password varchar(100) NOT NULL DEFAULT '' COMMENT '密码',
	created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;