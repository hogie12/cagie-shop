-- One database per service. Runs only when the postgres volume is first created.
CREATE DATABASE auth;
CREATE DATABASE catalog;
CREATE DATABASE inventory;
CREATE DATABASE orders;
CREATE DATABASE payment;
