CREATE DATABASE IF NOT EXISTS anzuchat;

USE anzuchat; 

CREATE TABLE IF NOT EXISTS users (
  id            INT AUTO_INCREMENT NOT NULL,
  username      VARCHAR(128) NOT NULL,
  PRIMARY KEY (`id`)
);

CREATE USER IF NOT EXISTS 'anzuserver'@'localhost' IDENTIFIED BY 'JNu6FYk62F3aLPmS9Np1f256MK946A45';

SELECT User FROM mysql.user;

GRANT ALL PRIVILEGES ON anzuchat.* TO 'anzuserver'@'localhost' WITH GRANT OPTION;

FLUSH PRIVILEGES;

SHOW GRANTS FOR 'anzuserver'@'localhost';