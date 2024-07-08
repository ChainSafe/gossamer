# pubip

A simple package for getting your public IP address by several services. 
It's inspired by https://github.com/chyeh/pubip.

Pubip validates the results from several services and returns the IP address if a valid one is found. 
Based on the assumption that the services your program depends on are not always available, 
it's better to have more backups services. This package gives you the public IP address from several different APIs.