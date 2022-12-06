//go:build integration

package dbrepo

import (
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"github.com/spacesedan/testing-course/webapp/pkg/repository"
	"log"
	"os"
	"testing"
	"time"
)

var (
	host     string = "localhost"
	user     string = "postgres"
	password string = "postgres"
	dbName   string = "users_test"
	port     string = "5435"
	dsn      string = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5"
)

var resource *dockertest.Resource
var pool *dockertest.Pool
var testDB *sql.DB
var testRepo repository.DataBaseRepo

func TestMain(m *testing.M) {
	// connect to docker; fail if docker not running
	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker; is it running? %s", err)
	}

	pool = p

	// set up our docker options, specifying the image and so forth
	opts := &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14.5",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}
	// get a resource
	resource, err = pool.RunWithOptions(opts)
	if err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not start resource: %s", err)
	}

	// start the image and wait until it's ready
	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbName))
		if err != nil {
			log.Println("Error: ", err)
			return err
		}
		return testDB.Ping()
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect to database: %s", err)
	}

	// populate the database with empty tables
	err = createTables()
	if err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect t odatabase: %s", err)
	}

	testRepo = &PostgresDBRepo{
		DB: testDB,
	}

	// run the tests
	code := m.Run()

	// cleanup
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	os.Exit(code)
}

func createTables() error {
	tableSQL, err := os.ReadFile("./testdata/users.sql")
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, err = testDB.Exec(string(tableSQL))
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func Test_pingDB(t *testing.T) {
	err := testDB.Ping()
	if err != nil {
		t.Error(err)
	}
}

func TestPostgresDBRepo_InsertUser(t *testing.T) {
	testUser := data.User{
		FirstName: "Admin",
		LastName:  "User",
		Email:     "admin@example.com",
		Password:  "secret",
		IsAdmin:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	id, err := testRepo.InsertUser(testUser)
	if err != nil {
		t.Errorf("InsertUser() returned an error: %s", err)
	}

	if id != 1 {
		t.Errorf("InsertUser() returned wrong id; expected 1, but got %d", id)
	}
}

func TestPostgresDBRepo_AllUsers(t *testing.T) {
	users, err := testRepo.AllUsers()
	if err != nil {
		t.Errorf("AllUsers() returned an error: %s", err)
	}

	if len(users) != 1 {
		t.Errorf("AllUsers() returned wrong size; expected 1, got %d", len(users))
	}

	testUser := data.User{
		FirstName: "Jack",
		LastName:  "Smith",
		Email:     "jack@smith.com",
		Password:  "secret",
		IsAdmin:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, _ = testRepo.InsertUser(testUser)

	users, err = testRepo.AllUsers()
	if err != nil {
		t.Errorf("AllUsers() returned an error: %s", err)
	}

	if len(users) != 2 {
		t.Errorf("AllUsers() returned wrong size after insert; expected 2, got %d", len(users))
	}
}

func TestPostgresDBRepo_GetUser(t *testing.T) {
	user, err := testRepo.GetUser(1)
	if err != nil {
		t.Errorf("error getting user by id: %s", err)
	}

	if user.Email != "admin@example.com" {
		t.Errorf("GetUser() returned wrong user with given email; expected admin@example.com, got %s", user.Email)
	}

	_, err = testRepo.GetUser(3)
	if err == nil {
		t.Error("no error reported when getting a non existant user by id.")
	}
}

func TestPostgresDBRepo_GetUserByEmail(t *testing.T) {
	user, err := testRepo.GetUserByEmail("jack@smith.com")
	if err != nil {
		t.Errorf("error getting user by id: %s", err)
	}

	if user.ID != 2 {
		t.Errorf("GetUserByEmail() returned wrong id; expected 2, got %d", user.ID)
	}
}

func TestPostgresDBRepo_UpdateUser(t *testing.T) {
	user, _ := testRepo.GetUser(2)
	user.FirstName = "Jane"
	user.Email = "jane@smith.com"

	err := testRepo.UpdateUser(*user)
	if err != nil {
		t.Errorf("error updating user %d: %s", 2, err)
	}

	user, _ = testRepo.GetUser(2)
	if user.FirstName != "Jane" || user.Email != "jane@smith.com" {
		t.Errorf("expected updated record to have first name Jane and email jane@smaith.com, but got %s %s", user.FirstName, user.Email)
	}
}

func TestPostgresDBRepo_DeleteUser(t *testing.T) {
	err := testRepo.DeleteUser(2)
	if err != nil {
		t.Errorf("error deleting user id 2; %s", err)
	}

	_, err = testRepo.GetUser(2)
	if err == nil {
		t.Error("retrieved user id 2, who should have been deleted")
	}
}

func TestPostgresDBRepo_ResetPassword(t *testing.T) {
	err := testRepo.ResetPassword(1, "password")
	if err != nil {
		t.Error("error resetting user's password; ", err)
	}

	user, _ := testRepo.GetUser(1)
	matches, err := user.PasswordMatches("password")
	if err != nil {
		t.Error(err)
	}
	if !matches {
		t.Errorf("password should match 'password' but does not")
	}

}

func TestPostgresDBRepo_InsertUserImage(t *testing.T) {
	userImage := data.UserImage{
		UserID:    1,
		FileName:  "test.jpg",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	newId, err := testRepo.InsertUserImage(userImage)
	if err != nil {
		t.Error("inserting user image failed ", err)
	}

	if newId != 1 {
		t.Error("got wrong id for image; should be 1 but got ", newId)
	}

	userImage.UserID = 100
	_, err = testRepo.InsertUserImage(userImage)
	if err == nil {
		t.Error("inserted an user image with non-existent user id", err)
	}
}
