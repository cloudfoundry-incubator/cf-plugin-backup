You can use the following json example. It will prompt for all 4 properties and tell you that 3 of them are required :
```
{  
   "type":"object",
   "properties":{  
      "server":{  
         "type":"string"
      },
      "port":{  
         "type":"integer"
      },
      "userid":{  
         "type":"integer"
      },
      "password":{  
         "type":"string"
      }
   },
   "required":[  
      "server",
      "port",
	  "password"
   ]
}
```
The output of schema parsing looks like this (with values entered by hand):
```
Insert string value for /server [required]> 127.0.0.1

Insert integer value for /port [required]> 8111

Insert integer value for /userid> 0

Insert string value for /password [required]>
```

Other possible types are "number" and "boolean".
