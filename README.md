# receipt-processor  

This repository hosts a solution to the [receipt-processor-challenge](https://github.com/fetch-rewards/receipt-processor-challenge) written in Go. To **get started**, you can first run:
```
go test
``` 
to verify that the receipt-processor code is working correctly. To see the tests, look to the `main_test.go` file. After running the tests, the command: 
```
go run . [OPTIONAL_PORT]
``` 
will start up the server on your machine on the port provided in the arguments (or on 8080 if no port is provided). There are also Postman integration tests in this repo. To run these tests, you first need to install the [Postman CLI](https://learning.postman.com/docs/postman-cli/postman-cli-installation/#mac-apple-silicon-installation). Following this link should give clear instructions on installation to choose based on your machine. After installing the CLI, verify that it has been installed using: 
```
postman -v
```
Next, two terminals should be opened. One should be used to start the server using the command above. For running these tests, you should leave the port argument blank to run on 8080. Finally, run the command: 
```
postman collection run ReceiptProcessorTests.postman_collection.json --environment ReceiptProcessorEnv.postman_environment.json
```
You should see all of the integration tests passing and green. If you can't run on port 8080, you can also run this flag to change the port that the tests are using: 
```
--env-var "PORT=[port-value]"
```

