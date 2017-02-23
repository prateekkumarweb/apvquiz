CREATE DATABASE apvquiz;
CREATE TABLE users (
	id int auto_increment,
	username varchar(180) not null unique,
	password varchar(180) not null,
	primary key (id)
);

CREATE TABLE questions (
	id int auto_increment,
	question text not null,
	option1 varchar(180) not null,
	option2 varchar(180) not null,
	option3 varchar(180) not null,
	option4 varchar(180) not null,
	answer int not null,
	primary key (id)
);