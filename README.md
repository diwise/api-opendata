# api-opendata

A service that manages catalogs and datasets for open data.

## enabling/disabling services

You control which services start by setting the `ENABLED_SERVICES` environment variable. It accepts a **comma-separated list** of service keys, or the special token `all`, which will turn on all services. If left empty, `ENABLED_SERVICES` will default to `all`.

In order to start specific services, add the appropriate service names to a comma separated list. The following services are currently available:
    
    - airqualities
    - beaches (enabling beaches also turns on the water quality service as it is required for beaches to run)
    - cityworks
    - exercisetrails
    - roadaccidents
    - sportsfields
    - sportsvenues
    - stratsys
    - traffic
    - waterqualities
    - weather

**example**
 ```bash
 export ENABLED_SERVICES="airqualities,cityworks,traffic"