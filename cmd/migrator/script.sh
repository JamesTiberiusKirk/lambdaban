#!/bin/sh


# exist if any are non 0 commands
set -e  

/app/migrator version 
/app/migrator schema-up 
/app/migrator migrate 
