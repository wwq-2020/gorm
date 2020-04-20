package testdata

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestOp(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		"root",
		"devilsm8875",
		"127.0.0.1",
		3306,
		"testdata",
		"parseTime=true",
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open db,err: %#v\r\n", err)
	}
	now := time.Now()
	repo := NewRepo(db)
	givenUser := &User{
		Name:      "user1",
		Password:  "password1",
		CreatedAt: now,
	}
	id, err := repo.Create(context.Background(), givenUser)
	if err != nil {
		t.Fatalf("failed to Create user,err: %#v\r\n", err)
	}
	if id <= 0 {
		t.Fatalf("unexpected id ,id: %#v\r\n", id)
	}
	givenUsers := []*User{
		&User{
			Name:      "user2",
			Password:  "password2",
			CreatedAt: now,
		},
		&User{
			Name:      "user3",
			Password:  "password3",
			CreatedAt: now,
		},
		&User{
			Name:      "user4",
			Password:  "password4",
			CreatedAt: now,
		},
	}

	if repo.BatchCreate(context.Background(), givenUsers); err != nil {
		t.Fatalf("failed to BatchCreate user,err: %#v\r\n", err)
	}

	gotUsers, err := repo.Find(context.Background(), nil, WithSorterBuilder(SortByID(true)), WithPaginate(0, 10))
	if err != nil {
		t.Fatalf("failed to Find user,err: %#v\r\n", err)
	}
	if len(gotUsers) != 4 {
		t.Fatalf("Find unexpected user count")
	}

	gotUser, err := repo.FindOne(context.Background(), NameEq("user1"))
	if err != nil {
		t.Fatalf("failed to FindOne user,err: %#v\r\n", err)
	}

	if gotUser.Name != givenUser.Name ||
		gotUser.Password != givenUser.Password {
	}

	rowsAffected, err := repo.Update(context.Background(), NameEq("user1"), Password("password2"))
	if err != nil {
		t.Fatalf("failed to Update user,err: %#v\r\n", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("Update unexpected rowsAffetced")
	}

	gotUsers, err = repo.Find(context.Background(), nil)
	if err != nil {
		t.Fatalf("failed to Find user,err: %#v\r\n", err)
	}

	if len(gotUsers) != 4 {
		t.Fatalf("Find unexpected user count")
	}

	gotUser, err = repo.FindOne(context.Background(), NameEq("user1"))
	if err != nil {
		t.Fatalf("failed to FindOne user,err: %#v\r\n", err)
	}
	if gotUser.Name != givenUser.Name ||
		gotUser.Password != "password2" {
	}

	rowsAffected, err = repo.Delete(context.Background(), NameEq("user1"))
	if err != nil {
		t.Fatalf("failed to Delete user,err: %#v\r\n", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("Delete unexpected rowsAffetced")
	}

	_, err = repo.FindOne(context.Background(), NameEq("user1"))
	if err != sql.ErrNoRows {
		t.Fatalf("failed to Find user,err: %#v\r\n", err)
	}

	rowsAffected, err = repo.Delete(context.Background(), nil)
	if err != nil {
		t.Fatalf("failed to Delete user,err: %#v\r\n", err)
	}
	if rowsAffected != 3 {
		t.Fatalf("Delete unexpected rowsAffetced")
	}
	gotUsers, err = repo.Find(context.Background(), nil)
	if err != nil {
		t.Fatalf("failed to Find user,err: %#v\r\n", err)
	}
	if len(gotUsers) != 0 {
		t.Fatalf("Find unexpected user count")
	}
}
