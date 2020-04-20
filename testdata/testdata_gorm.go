package testdata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TxHandler TxHandler
type TxHandler func(ctx context.Context, tx Tx) error

// Repo Repo
type Repo interface {
	InTx(ctx context.Context, txHandler TxHandler) error
	Tx
}

// Tx Tx
type Tx interface {
	Find(ctx context.Context, filter Filter, opts ...Option) ([]*User, error)
	FindOne(ctx context.Context, filter Filter, opts ...Option) (*User, error)
	Delete(ctx context.Context, filter Filter) (int64, error)
	Update(ctx context.Context, filter Filter, updaters ...Updater) (int64, error)
	Create(ctx context.Context, obj *User) (int64, error)
	BatchCreate(ctx context.Context, objs []*User) error
}

type sqlCommon interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, ags ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type sqlDB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type repo struct {
	tx
}

type tx struct {
	db sqlCommon
}

// NewUserRepo NewUserRepo
func NewRepo(db *sql.DB) Repo {
	return &repo{
		tx{db: db},
	}
}

func (rp repo) InTx(ctx context.Context, txHandler TxHandler) error {
	db, ok := rp.db.(sqlDB)
	if !ok {
		return errors.New("do not support tx")
	}
	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer dbTx.Rollback()
	tx := &tx{dbTx}
	if err := txHandler(ctx, tx); err != nil {
		return err
	}
	if err := dbTx.Commit(); err != nil {
		return err
	}
	return nil
}

// Find Find
func (tx tx) Find(ctx context.Context, filter Filter, opts ...Option) ([]*User, error) {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}
	sortStr := ""
	if options.sorterBuilder != nil {
		sortStr = fmt.Sprintf(" order by %s ", options.sorterBuilder.Build())
	}

	paginate := ""
	if options.paginate != nil {
		paginate = fmt.Sprintf(" inner join (select id from user %s limit %d, %d) tmp on user.id = tmp.id ", sortStr, options.paginate.offset, options.paginate.size)
		if filter != nil && filter.Cond() != "" {
			paginate = fmt.Sprintf(" inner join (select id from user where %s %s limit %d, %d) tmp on user.id = tmp.id ", filter.Cond(), sortStr, options.paginate.offset, options.paginate.size)
		}
	}

	withLock := ""
	if options.withLock {
		withLock = " for update "
	}

	var rows *sql.Rows
	var err error
	if filter == nil || filter.Cond() == "" {
		sql := fmt.Sprintf("select user.id,name,password,created_at from user%s%s%s", sortStr, paginate, withLock)
		if paginate != "" {
			sql = fmt.Sprintf("select user.id,name,password,created_at from user%s%s", paginate, withLock)
		}
		rows, err = tx.db.QueryContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("select user.id,name,password,created_at from user where %s%s%s%s", filter.Cond(), sortStr, paginate, withLock)
		if paginate != "" {
			sql = fmt.Sprintf("select user.id,name,password,created_at from user %s%s", paginate, withLock)
		}
		rows, err = tx.db.QueryContext(ctx, sql, filter.Args()...)
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
func (tx tx) FindOne(ctx context.Context, filter Filter, opts ...Option) (*User, error) {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	sortStr := ""
	if options.sorterBuilder != nil {
		sortStr = fmt.Sprintf(" order by %s ", options.sorterBuilder.Build())
	}

	paginate := ""
	if options.paginate != nil {
		paginate = fmt.Sprintf(" limit %d, %d ", options.paginate.offset, options.paginate.size)
	}

	withLock := ""
	if options.withLock {
		withLock = " for update "
	}

	var row *sql.Row
	if filter == nil || filter.Cond() == "" {
		sql := fmt.Sprintf("select user.id,name,password,created_at from user%s%s%s", sortStr, paginate, withLock)
		if paginate != "" {
			sql = fmt.Sprintf("select user.id,name,password,created_at from user%s%s", paginate, withLock)
		}
		row = tx.db.QueryRowContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("select user.id,name,password,created_at from user where %s%s%s%s", filter.Cond(), sortStr, paginate, withLock)
		if paginate != "" {
			sql = fmt.Sprintf("select user.id,name,password,created_at from user %s%s", paginate, withLock)
		}
		row = tx.db.QueryRowContext(ctx, sql, filter.Args()...)
	}
	result := &User{}
	if err := row.Scan(&result.ID, &result.Name, &result.Password, &result.CreatedAt); err != nil {
		return nil, err
	}
	return result, nil
}

// Delete Delete
func (tx tx) Delete(ctx context.Context, filter Filter) (int64, error) {
	var result sql.Result
	var err error
	if filter == nil || filter.Cond() == "" {
		result, err = tx.db.ExecContext(ctx, "delete from user")
	} else {
		sql := fmt.Sprintf("delete from user where %s", filter.Cond())
		result, err = tx.db.ExecContext(ctx, sql, filter.Args()...)
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
func (tx tx) Update(ctx context.Context, filter Filter, updaters ...Updater) (int64, error) {
	var result sql.Result
	var err error
	updateStrs := make([]string, 0, len(updaters))
	updateArgs := make([]interface{}, 0, len(updaters))
	for _, updater := range updaters {
		updateStrs = append(updateStrs, updater.Set())
		updateArgs = append(updateArgs, updater.Arg())
	}
	if filter == nil || filter.Cond() == "" {
		sqlBaseStr := "update user set %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs, ","))
		result, err = tx.db.ExecContext(ctx, sqlStr, updateArgs...)
	} else {
		sqlBaseStr := "update user set %s where %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs, ","), filter.Cond())
		sqlArgs := append(updateArgs, filter.Args()...)
		result, err = tx.db.ExecContext(ctx, sqlStr, sqlArgs...)
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
func (tx tx) Create(ctx context.Context, obj *User) (int64, error) {
	result, err := tx.db.ExecContext(ctx, "insert into user(name,password,created_at) values (?,?,?)", obj.Name, obj.Password, obj.CreatedAt)
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
func (tx tx) BatchCreate(ctx context.Context, objs []*User) error {
	sqlBaseStr := "insert into user(name,password,created_at) values %s"
	sqlPlaceHolder := make([]string, 0, len(objs))
	sqlArgs := make([]interface{}, 0, len(objs)*4)
	for _, obj := range objs {
		sqlPlaceHolder = append(sqlPlaceHolder, "(?,?,?)")
		sqlArgs = append(sqlArgs, obj.Name, obj.Password, obj.CreatedAt)
	}
	sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(sqlPlaceHolder, ","))
	if _, err := tx.db.ExecContext(ctx, sqlStr, sqlArgs...); err != nil {
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
	Or(...Filter) JoinableFilter
	And(...Filter) JoinableFilter
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

func (f *filter) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(f.Args()))
	conds = append(conds, f.Cond())
	args = append(args, f.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

func (f *filter) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(f.Args()))
	conds = append(conds, f.Cond())
	args = append(args, f.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
	}
}

type options struct {
	sorterBuilder SorterBuilder
	paginate      *paginate
	withLock      bool
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

func WithLock() Option {
	return func(o *options) {
		o.withLock = true
	}
}

// WithJoinSorterBuilders WithJoinSorterBuilders
func WithJoinSorterBuilder(joinSorterBuilders ...JoinableSorterBuilder) Option {
	return func(o *options) {
		result := joinSorterBuilders[0]
		for _, joinSorterBuilder := range joinSorterBuilders[1:] {
			result = result.Join(joinSorterBuilder)
		}
		o.sorterBuilder = result
	}
}

// Sorter Sorter
type Sorter string

// SorterBuilder SorterBuilder
type SorterBuilder interface {
	Build() string
}

// Join Join
func (s Sorter) Join(sorterBuilders ...SorterBuilder) JoinableSorterBuilder {
	result := string(s)
	for _, sorterBuilder := range sorterBuilders {
		result += "," + sorterBuilder.Build()
	}
	return Sorter(result)
}

// Build Build
func (s Sorter) Build() string {
	return string(s)
}

// JoinableSorterBuilder JoinableSorterBuilder
type JoinableSorterBuilder interface {
	SorterBuilder
	Join(...SorterBuilder) JoinableSorterBuilder
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
func (n IDEq) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDEq) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n IDNE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDNE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n IDBt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDBt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n IDLt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDLt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n IDBE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDBE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n IDLE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDLE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n IDIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n IDNotIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n IDNotIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameEq) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameEq) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameNE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameNE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameBt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameBt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameLt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameLt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameBE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameBE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameLE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameLE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n NameNotIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n NameNotIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordEq) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordEq) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordNE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordNE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordBt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordBt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordLt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordLt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordBE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordBE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordLE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordLE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n PasswordNotIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n PasswordNotIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtEq) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtEq) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtNE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtNE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtBt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtBt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtLt) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtLt) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtBE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtBE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtLE) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtLE) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
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
func (n CreatedAtNotIn) And(ands ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ands)+1)
	args := make([]interface{}, 0, len(ands)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, and := range ands {
		conds = append(conds, and.Cond())
		args = append(args, and.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "and")),
		args: args,
	}
}

// Or Or
func (n CreatedAtNotIn) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors {
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
	}
}

// SortByCreatedAt SortByCreatedAt
func SortByCreatedAt(asc bool) JoinableSorterBuilder {
	if asc {
		return Sorter("created_at asc")
	}
	return Sorter("created_at desc")
}
