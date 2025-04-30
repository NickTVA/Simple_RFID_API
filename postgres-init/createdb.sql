create database rfid;

\c rfid
create table tags(username text, tag text, expire timestamp);
