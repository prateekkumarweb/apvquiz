CREATE DATABASE apvquiz;
CREATE TABLE users (
	id int auto_increment,
	username varchar(180) not null unique,
	password varchar(180) not null,
	primary key (id)
);