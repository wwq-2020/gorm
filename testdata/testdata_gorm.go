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
func (rp UserRepo) Find(ctx context.Context, filter Filter, opts ...Option) ([]*User, error) {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}
	sortStr := " "
	if options.sorterBuilder != nil {
		sortStr = fmt.Sprintf(" order by %s", options.sorterBuilder.Build())
	}

	paginate := " "
	if options.paginate != nil {
		paginate = fmt.Sprintf(" limit %d, %d", options.paginate.offset, options.paginate.size)
	}

	var rows *sql.Rows
	var err error
	if filter.Cond() == "" {
		sql := fmt.Sprintf("select id,name,password,created_at from user%s%s", sortStr, paginate)
		rows, err = rp.db.QueryContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("select id,name,password,created_at from user where %s%s%s", filter.Cond(), sortStr, paginate)
		rows, err = rp.db.QueryContext(ctx, sql, filter.Args()...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*User
	for rows.Next() {
		result := &User{}
		if err := rows.Scan(&result.ID, &result.Name, &result.Password, &result.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// FindOne FindOne
func (rp UserRepo) FindOne(ctx context.Context, filter Filter, opts ...Option) (*User, error) {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	sortStr := " "
	if options.sorterBuilder != nil {
		sortStr = fmt.Sprintf(" order by %s", options.sorterBuilder.Build())
	}

	paginate := " "
	if options.paginate != nil {
		paginate = fmt.Sprintf(" limit %d, %d", options.paginate.offset, options.paginate.size)
	}

	var row *sql.Row
	if filter.Cond() == "" {
		sql := fmt.Sprintf("select id,name,password,created_at from user%s%s", sortStr, paginate)
		row = rp.db.QueryRowContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("select id,name,password,created_at from user where %s%s%s", filter.Cond(), sortStr, paginate)
		row = rp.db.QueryRowContext(ctx, sql, filter.Args()...)
	}
	result := &User{}
	if err := row.Scan(&result.ID, &result.Name, &result.Password, &result.CreatedAt); err != nil {
		return nil, err
	}
	return result, nil
}

// Delete Delete
func (rp UserRepo) Delete(ctx context.Context, filter Filter) (int64, error) {
	var result sql.Result
	var err error
	if filter.Cond() == "" {
		result, err = rp.db.ExecContext(ctx, "delete from user")
	} else {
		sql := fmt.Sprintf("delete from user where %s", filter.Cond())
		result, err = rp.db.ExecContext(ctx, sql, filter.Args()...)
	}
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
func (rp UserRepo) Update(ctx context.Context, filter Filter, updaters ...Updater) (int64, error) {
	var result sql.Result
	var err error
	updateStrs := make([]string, 0, len(updaters))
	updateArgs := make([]interface{}, 0, len(updaters))
	for _, updater := range updaters {
		updateStrs = append(updateStrs, updater.Set())
		updateArgs = append(updateArgs, updater.Arg())
	}
	if filter.Cond() == "" {
		sqlBaseStr := "update user set %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs, ","))
		result, err = rp.db.ExecContext(ctx, sqlStr, updateArgs...)
	} else {
		sqlBaseStr := "update user set %s where %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs, ","), filter.Cond())
		sqlArgs := append(updateArgs, filter.Args())
		result, err = rp.db.ExecContext(ctx, sqlStr, sqlArgs...)
	}
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
	result, err := rp.db.ExecContext(ctx, "insert into user(id,name,password,created_at) (?,?,?,?)", obj.ID, obj.Name, obj.Password, obj.CreatedAt)
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
	sqlBaseStr := "insert into user(id,name,password,created_at) %s"
	sqlPlaceHolder := make([]string, 0, len(objs))
	sqlArgs := make([]interface{}, 0, len(objs)*4)
	for _, obj := range objs {
		sqlPlaceHolder = append(sqlPlaceHolder, "(?,?,?,?)")
		sqlArgs = append(sqlArgs, obj.ID, obj.Name, obj.Password, obj.CreatedAt)
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
}

// JoinableFilter JoinableFilter
type JoinableFilter interface {
	Filter
	Or(Filter) JoinableFilter
	And(Filter) JoinableFilter
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

func (f *filter) And(and Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", f.Cond(), ") and (", and.Cond(), ")"}, ""),
		args: append(f.Args(), and.Args()...),
	}
}

func (f *filter) Or(or Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", f.Cond(), ") or (", or.Cond(), ")"}, ""),
		args: append(f.Args(), or.Args()...),
	}
}

type options struct {
	sorterBuilder SorterBuilder
	paginate      *paginate
}

type paginate struct {
	offset int64
	size   int
}

// Option Option
type Option func(*options)

// WithPaginate WithPaginate
func WithPaginate(offset int64, size int) Option {
	return func(o *options) {
		curPaginate := o.paginate
		if curPaginate == nil {
			curPaginate = &paginate{}
			o.paginate = curPaginate
		}
		curPaginate.offset = offset
		curPaginate.size = size
	}
}

// WithSorterBuilder WithSorterBuilder
func WithSorterBuilder(sorterBuilder SorterBuilder) Option {
	return func(o *options) {
		o.sorterBuilder = sorterBuilder
	}
}

// Sorter Sorter
type Sorter string

// SorterBuilder SorterBuilder
type SorterBuilder interface {
	Build() string
}

// Join Join
func (s Sorter) Join(sorterBuilder SorterBuilder) JoinableSorterBuilder {
	return Sorter(string(s) + "," + sorterBuilder.Build())
}

// Build Build
func (s Sorter) Build() string {
	return string(s)
}

// JoinableSorterBuilder JoinableSorterBuilder
type JoinableSorterBuilder interface {
	SorterBuilder
	Join(SorterBuilder) JoinableSorterBuilder
}

// ID ID
type ID int64

// Set Set
func (n ID) Set() string {
	return "id=?"
}

// Arg Arg
func (n ID) Arg() interface{} {
	return n
}

// IDEq IDEq
type IDEq int64

// Cond Cond
func (n IDEq) Cond() string {
	return "id=?"
}

// Args Args
func (n IDEq) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n IDEq) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDEq) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// IDNE IDNE
type IDNE int64

// Cond Cond
func (n IDNE) Cond() string {
	return "id != ?"
}

// Args Args
func (n IDNE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n IDNE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDNE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// IDBt IDBt
type IDBt int64

// Cond Cond
func (n IDBt) Cond() string {
	return "id>?"
}

// Args Args
func (n IDBt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n IDBt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDBt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// IDLt IDLt
type IDLt int64

// Cond Cond
func (n IDLt) Cond() string {
	return "id<?"
}

// Args Args
func (n IDLt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n IDLt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDLt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// IDBE IDBE
type IDBE int64

// Cond Cond
func (n IDBE) Cond() string {
	return "id>=?"
}

// Args Args
func (n IDBE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n IDBE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDBE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// IDLE IDLE
type IDLE int64

// Cond Cond
func (n IDLE) Cond() string {
	return "id<=?"
}

// Args Args
func (n IDLE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n IDLE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDLE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// IDIn IDIn
type IDIn []int64

// Cond Cond
func (n IDIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("id in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n IDIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n IDIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// IDNotIn IDNotIn
type IDNotIn []int64

// Cond Cond
func (n IDNotIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("id not in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n IDNotIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n IDNotIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n IDNotIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// SortByID SortByID
func SortByID(asc bool) JoinableSorterBuilder {
	if asc {
		return Sorter("id asc")
	}
	return Sorter("id desc")
}

// Name Name
type Name string

// Set Set
func (n Name) Set() string {
	return "name=?"
}

// Arg Arg
func (n Name) Arg() interface{} {
	return n
}

// NameEq NameEq
type NameEq string

// Cond Cond
func (n NameEq) Cond() string {
	return "name=?"
}

// Args Args
func (n NameEq) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n NameEq) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameEq) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// NameNE NameNE
type NameNE string

// Cond Cond
func (n NameNE) Cond() string {
	return "name != ?"
}

// Args Args
func (n NameNE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n NameNE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameNE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// NameBt NameBt
type NameBt string

// Cond Cond
func (n NameBt) Cond() string {
	return "name>?"
}

// Args Args
func (n NameBt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n NameBt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameBt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// NameLt NameLt
type NameLt string

// Cond Cond
func (n NameLt) Cond() string {
	return "name<?"
}

// Args Args
func (n NameLt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n NameLt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameLt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// NameBE NameBE
type NameBE string

// Cond Cond
func (n NameBE) Cond() string {
	return "name>=?"
}

// Args Args
func (n NameBE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n NameBE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameBE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// NameLE NameLE
type NameLE string

// Cond Cond
func (n NameLE) Cond() string {
	return "name<=?"
}

// Args Args
func (n NameLE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n NameLE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameLE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// NameIn NameIn
type NameIn []string

// Cond Cond
func (n NameIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("name in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n NameIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n NameIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// NameNotIn NameNotIn
type NameNotIn []string

// Cond Cond
func (n NameNotIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("name not in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n NameNotIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n NameNotIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n NameNotIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// SortByName SortByName
func SortByName(asc bool) JoinableSorterBuilder {
	if asc {
		return Sorter("name asc")
	}
	return Sorter("name desc")
}

// Password Password
type Password string

// Set Set
func (n Password) Set() string {
	return "password=?"
}

// Arg Arg
func (n Password) Arg() interface{} {
	return n
}

// PasswordEq PasswordEq
type PasswordEq string

// Cond Cond
func (n PasswordEq) Cond() string {
	return "password=?"
}

// Args Args
func (n PasswordEq) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n PasswordEq) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordEq) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// PasswordNE PasswordNE
type PasswordNE string

// Cond Cond
func (n PasswordNE) Cond() string {
	return "password != ?"
}

// Args Args
func (n PasswordNE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n PasswordNE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordNE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// PasswordBt PasswordBt
type PasswordBt string

// Cond Cond
func (n PasswordBt) Cond() string {
	return "password>?"
}

// Args Args
func (n PasswordBt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n PasswordBt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordBt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// PasswordLt PasswordLt
type PasswordLt string

// Cond Cond
func (n PasswordLt) Cond() string {
	return "password<?"
}

// Args Args
func (n PasswordLt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n PasswordLt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordLt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// PasswordBE PasswordBE
type PasswordBE string

// Cond Cond
func (n PasswordBE) Cond() string {
	return "password>=?"
}

// Args Args
func (n PasswordBE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n PasswordBE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordBE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// PasswordLE PasswordLE
type PasswordLE string

// Cond Cond
func (n PasswordLE) Cond() string {
	return "password<=?"
}

// Args Args
func (n PasswordLE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n PasswordLE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordLE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// PasswordIn PasswordIn
type PasswordIn []string

// Cond Cond
func (n PasswordIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("password in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n PasswordIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n PasswordIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// PasswordNotIn PasswordNotIn
type PasswordNotIn []string

// Cond Cond
func (n PasswordNotIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("password not in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n PasswordNotIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n PasswordNotIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n PasswordNotIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// SortByPassword SortByPassword
func SortByPassword(asc bool) JoinableSorterBuilder {
	if asc {
		return Sorter("password asc")
	}
	return Sorter("password desc")
}

// CreatedAt CreatedAt
type CreatedAt time.Time

// Set Set
func (n CreatedAt) Set() string {
	return "created_at=?"
}

// Arg Arg
func (n CreatedAt) Arg() interface{} {
	return n
}

// CreatedAtEq CreatedAtEq
type CreatedAtEq time.Time

// Cond Cond
func (n CreatedAtEq) Cond() string {
	return "created_at=?"
}

// Args Args
func (n CreatedAtEq) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n CreatedAtEq) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtEq) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// CreatedAtNE CreatedAtNE
type CreatedAtNE time.Time

// Cond Cond
func (n CreatedAtNE) Cond() string {
	return "created_at != ?"
}

// Args Args
func (n CreatedAtNE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n CreatedAtNE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtNE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// CreatedAtBt CreatedAtBt
type CreatedAtBt time.Time

// Cond Cond
func (n CreatedAtBt) Cond() string {
	return "created_at>?"
}

// Args Args
func (n CreatedAtBt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n CreatedAtBt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtBt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// CreatedAtLt CreatedAtLt
type CreatedAtLt time.Time

// Cond Cond
func (n CreatedAtLt) Cond() string {
	return "created_at<?"
}

// Args Args
func (n CreatedAtLt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n CreatedAtLt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtLt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// CreatedAtBE CreatedAtBE
type CreatedAtBE time.Time

// Cond Cond
func (n CreatedAtBE) Cond() string {
	return "created_at>=?"
}

// Args Args
func (n CreatedAtBE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n CreatedAtBE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtBE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// CreatedAtLE CreatedAtLE
type CreatedAtLE time.Time

// Cond Cond
func (n CreatedAtLE) Cond() string {
	return "created_at<=?"
}

// Args Args
func (n CreatedAtLE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n CreatedAtLE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtLE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// CreatedAtIn CreatedAtIn
type CreatedAtIn []time.Time

// Cond Cond
func (n CreatedAtIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("created_at in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n CreatedAtIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n CreatedAtIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// CreatedAtNotIn CreatedAtNotIn
type CreatedAtNotIn []time.Time

// Cond Cond
func (n CreatedAtNotIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("created_at not in (%s)", strings.Join(placeHolders, ","))
}

// Args Args
func (n CreatedAtNotIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n CreatedAtNotIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n CreatedAtNotIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// SortByCreatedAt SortByCreatedAt
func SortByCreatedAt(asc bool) JoinableSorterBuilder {
	if asc {
		return Sorter("created_at asc")
	}
	return Sorter("created_at desc")
}
