all : install

clean :
	@echo ">>> Cleaning and initializing hastycsv project <<<"
	@go clean
	@gofmt -w .

test : clean
	@echo ">>> Running unit tests <<<"
	@go test

test-coverage : clean
	@echo ">>> Running unit tests and calculating code coverage <<<"
	@go test -cover

install : test
	@echo ">>> Building and installing hastycsv <<<"
	@go install
	@echo ">>> hastycsv installed successfully! <<<"
	@echo ""
