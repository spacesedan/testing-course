# if we ever get an permission denied when trying to run tests use command:
#		sudo chmod -R 777 [name_of_directory]

coverage:
	go test -v -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html