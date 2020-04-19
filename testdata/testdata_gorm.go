package testdata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// UserRepo UserRepo
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo NewUserRepo
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

// Find Find
func (rp UserRepo) Find(ctx context.Context, filter Filter) ([]*User, error) {
	sql := fmt.Sprintf("select id,name,password from user where %s", filter.Cond())
	rows, err := rp.db.QueryContext(ctx, sql, filter.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*User
	for rows.Next() {
		result := &User{}
		if err := rows.Scan(&result.ID, &result.Name, &result.Password); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// Delete Delete
func (rp UserRepo) Delete(ctx context.Context, filter Filter) (int64, error) {
	sql := fmt.Sprintf("delete from user where %s", filter.Cond())
	result, err := rp.db.ExecContext(ctx, sql, filter.Args()...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

// Update Update
func (rp UserRepo) Update(ctx context.Context, filter Filter, updates ...Updater) (int64, error) {
	updateStrs := make([]string, 0, len(updates))
	updateArgs := make([]interface{}, 0, len(updates))
	for _, update := range updates {
		updateStrs = append(updateStrs, update.Set())
		updateArgs = append(updateArgs, update.Arg())
	}
	sqlBaseStr := "update user set  %s where %s"
	sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs, ","), filter.Cond())
	sqlArgs := append(updateArgs, filter.Args())
	result, err := rp.db.ExecContext(ctx, sqlStr, sqlArgs...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

// Create Create
func (rp UserRepo) Create(ctx context.Context, obj *User) (int64, error) {
	result, err := rp.db.ExecContext(ctx, "insert into user(id,name,password) (?,?,?)", obj.ID, obj.Name, obj.Password)
	if err != nil {
		return 0, err
	}
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return lastInsertID, nil
}

// BatchCreate BatchCreate
func (rp UserRepo) BatchCreate(ctx context.Context, objs []*User) error {
	sqlBaseStr := "insert into user(id,name,password) %s"
	sqlPlaceHolder := make([]string, 0, len(objs))
	sqlArgs := make([]interface{}, 0, len(objs)*3)
	for _, obj := range objs {
		sqlPlaceHolder = append(sqlPlaceHolder, "(?,?,?)")
		sqlArgs = append(sqlArgs, obj.ID, obj.Name, obj.Password)
	}
	sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(sqlPlaceHolder, ","))
	if _, err := rp.db.ExecContext(ctx, sqlStr, sqlArgs...); err != nil {
		return err
	}
	return nil
}

// Updater Updater
type Updater interface {
	Set() string
	Arg() interface{}
}

// Filter Filter
type Filter interface {
	Cond() string
	Args() []interface{}
	Or(Filter) Filter
	And(Filter) Filter
}

type filter struct {
	cond string
	args []interface{}
}

func (f *filter) Cond() string {
	return f.cond
}

func (f *filter) Args() []interface{} {
	return f.args
}

func (f *filter) And(and Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", f.Cond(), ") and (", and.Cond(), ")"}, ""),
		args: append(f.Args(), and.Args()...),
	}
}

func (f *filter) Or(or Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", f.Cond(), ") or (", or.Cond(), ")"}, ""),
		args: append(f.Args(), or.Args()...),
	}
}

// ID ID
type ID int64

// Cond Cond
func (n ID) Cond() string {
	return "id=?"
}

// Args Args
func (n ID) Args() []interface{} {
	return []interface{}{n}
}

// Set Set
func (n ID) Set() string {
	return "id=?"
}

// Arg Arg
func (n ID) Arg() interface{} {
	return n
}

// And And
func (n ID) And(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n ID) Or(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Name Name
type Name string

// Cond Cond
func (n Name) Cond() string {
	return "name=?"
}

// Args Args
func (n Name) Args() []interface{} {
	return []interface{}{n}
}

// Set Set
func (n Name) Set() string {
	return "name=?"
}

// Arg Arg
func (n Name) Arg() interface{} {
	return n
}

// And And
func (n Name) And(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n Name) Or(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Password Password
type Password string

// Cond Cond
func (n Password) Cond() string {
	return "password=?"
}

// Args Args
func (n Password) Args() []interface{} {
	return []interface{}{n}
}

// Set Set
func (n Password) Set() string {
	return "password=?"
}

// Arg Arg
func (n Password) Arg() interface{} {
	return n
}

// And And
func (n Password) And(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n Password) Or(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}
