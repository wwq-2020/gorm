package main

const tplStr = `
// TxHandler TxHandler
type TxHandler func(ctx context.Context, tx Tx) error

// Repo Repo
type Repo interface {
	InTx(ctx context.Context, txHandler TxHandler) error
	Tx
}

// Tx Tx
type Tx interface {
	Find(ctx context.Context, filter Filter, opts ...Option) ([]*{{.Name}}, error)
	FindOne(ctx context.Context, filter Filter, opts ...Option) (*{{.Name}}, error)
	Delete(ctx context.Context, filter Filter) (int64, error)
	Update(ctx context.Context, filter Filter, updaters ...Updater) (int64, error)
	Create(ctx context.Context,obj *{{.Name}}) (int64, error)
	BatchCreate(ctx context.Context, objs []*{{.Name}}) error
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

// New{{.Name}}Repo New{{.Name}}Repo
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
func (tx tx) Find(ctx context.Context, filter Filter, opts ...Option) ([]*{{.Name}}, error) {
	options:=&options{}
	for _,opt := range opts {
		opt(options)
	}
	sortStr := ""
	if options.sorterBuilder != nil {
		sortStr = fmt.Sprintf(" order by %s ", options.sorterBuilder.Build())
	}

	paginate := ""
	if options.paginate != nil {
		paginate = fmt.Sprintf(" inner join ({{.PaginateFindSQL}} %s limit %d, %d) tmp on {{.Tablename}}.id = tmp.id ", sortStr, options.paginate.offset, options.paginate.size)
		if filter != nil && filter.Cond() != "" {
			paginate = fmt.Sprintf(" inner join ({{.PaginateFindSQL}} where %s %s limit %d, %d) tmp on {{.Tablename}}.id = tmp.id ", filter.Cond(), sortStr, options.paginate.offset, options.paginate.size)
		}
	}

	withLock := ""
	if options.withLock {
		withLock = " for update "
	}

	var rows *sql.Rows
	var err error
	if filter == nil || filter.Cond() == "" {
		sql :=  fmt.Sprintf("{{.FindSQL}}%s%s%s", sortStr, paginate, withLock)
		if paginate != "" {
			sql =  fmt.Sprintf("{{.FindSQL}}%s%s", paginate, withLock)
		}
		rows, err = tx.db.QueryContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("{{.FindSQL}} where %s%s%s%s", filter.Cond(), sortStr, paginate, withLock)
		if paginate != "" {
			sql = fmt.Sprintf("{{.FindSQL}} %s%s", paginate, withLock)
		}
		rows, err = tx.db.QueryContext(ctx, sql, filter.Args()... )
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*{{.Name}}
	for rows.Next() {
		result := &{{.Name}}{}
		if err := rows.Scan({{.Scan|raw}}); err !=nil {
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
func (tx tx) FindOne(ctx context.Context, filter Filter, opts ...Option) (*{{.Name}}, error) {
	options:=&options{}
	for _,opt := range opts {
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
		sql :=  fmt.Sprintf("{{.FindSQL}}%s%s%s", sortStr, paginate, withLock)
		if paginate != "" {
			sql =  fmt.Sprintf("{{.FindSQL}}%s%s", paginate, withLock)
		}
		row= tx.db.QueryRowContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("{{.FindSQL}} where %s%s%s%s", filter.Cond(), sortStr, paginate, withLock)
		if paginate != "" {
			sql = fmt.Sprintf("{{.FindSQL}} %s%s", paginate, withLock)
		}
		row= tx.db.QueryRowContext(ctx, sql, filter.Args()... )
	}
	result := &{{.Name}}{}
	if err := row.Scan({{.Scan|raw}}); err !=nil {
		return nil, err
	}
	return result, nil
}

// Delete Delete
func (tx tx) Delete(ctx context.Context, filter Filter) (int64, error){
	var result sql.Result
	var err error
	if filter == nil || filter.Cond() == "" {
		result, err = tx.db.ExecContext(ctx, "{{.DeleteSQL}}")
	} else {
		sql:=fmt.Sprintf("{{.DeleteSQL}} where %s", filter.Cond())
		result, err = tx.db.ExecContext(ctx, sql, filter.Args()... )
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
func (tx tx) Update(ctx context.Context, filter Filter, updaters ...Updater) (int64, error){
	var result sql.Result
	var err error
	updateStrs := make([]string, 0, len(updaters))
	updateArgs := make([]interface{}, 0, len(updaters))
	for _, updater := range updaters {
		updateStrs = append(updateStrs, updater.Set())
		updateArgs = append(updateArgs, updater.Arg())
	}
	if filter == nil || filter.Cond() == "" {
		sqlBaseStr := "{{.UpdateSQL}} %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs,","))
		result, err = tx.db.ExecContext(ctx, sqlStr, updateArgs...)
	} else {
		sqlBaseStr := "{{.UpdateSQL}} %s where %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs,","), filter.Cond())
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
func (tx tx) Create(ctx context.Context,obj *{{.Name}}) (int64, error) {
	result, err := tx.db.ExecContext(ctx, "{{.CreateSQL}} ({{.CreatePlaceHolder}})", {{.CreateValue}})
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
func (tx tx) BatchCreate(ctx context.Context, objs []*{{.Name}}) error {
	sqlBaseStr := "{{.CreateSQL}} %s"
	sqlPlaceHolder := make([]string, 0, len(objs))
	sqlArgs := make([]interface{}, 0, len(objs)*{{.ColumnCount}})
	for _, obj := range objs {
		sqlPlaceHolder = append(sqlPlaceHolder, "({{.CreatePlaceHolder}})")
		sqlArgs = append(sqlArgs, {{.CreateValue}})
	}
	sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(sqlPlaceHolder, ","))
	if _,err := tx.db.ExecContext(ctx, sqlStr, sqlArgs...); err != nil {
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
type JoinableFilter interface  {
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
	paginate *paginate
	withLock bool
}

type paginate struct{
	offset int64
	size int
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
func WithSorterBuilder(sorterBuilder SorterBuilder) Option{
	return func(o *options) {
		o.sorterBuilder = sorterBuilder
	}
}

func WithLock() Option{
	return func(o *options) {
		o.withLock = true
	}
}

// WithJoinSorterBuilders WithJoinSorterBuilders
func WithJoinSorterBuilder(joinSorterBuilders ...JoinableSorterBuilder) Option{
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

{{range $idx,$each := .Fields}}

// {{$each.Name}} {{$each.Name}}
type {{$each.Name}} {{$each.Type}}
// Set Set
func (n {{$each.Name}}) Set() string {
	return "{{$each.Column}}=?"
}

// Arg Arg
func (n {{$each.Name}}) Arg() interface{} {
	return n
}

// {{$each.Name}}Eq {{$each.Name}}Eq
type {{$each.Name}}Eq {{$each.Type}}

// Cond Cond
func (n {{$each.Name}}Eq) Cond() string {
	return "{{$each.Column}}=?"
}

// Args Args
func (n {{$each.Name}}Eq) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n {{$each.Name}}Eq) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}Eq) Or(ors ...Filter) JoinableFilter {
	conds := make([]string, 0, len(ors)+1)
	args := make([]interface{}, 0, len(ors)+len(n.Args()))
	conds = append(conds, n.Cond())
	args = append(args, n.Args())
	for _, or := range ors{
		conds = append(conds, or.Cond())
		args = append(args, or.Args())
	}
	return &filter{
		cond: fmt.Sprintf("(%s)", strings.Join(conds, "or")),
		args: args,
	}
}

// {{$each.Name}}NE {{$each.Name}}NE
type {{$each.Name}}NE {{$each.Type}}

// Cond Cond
func (n {{$each.Name}}NE) Cond() string {
	return "{{$each.Column}} != ?"
}

// Args Args
func (n {{$each.Name}}NE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n {{$each.Name}}NE) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}NE) Or(ors ...Filter) JoinableFilter {
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

// {{$each.Name}}Bt {{$each.Name}}Bt
type {{$each.Name}}Bt {{$each.Type}}

// Cond Cond
func (n {{$each.Name}}Bt) Cond() string {
	return "{{$each.Column}}{{$.Bt|raw}}?"
}

// Args Args
func (n {{$each.Name}}Bt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n {{$each.Name}}Bt) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}Bt) Or(ors ...Filter) JoinableFilter {
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

// {{$each.Name}}Lt {{$each.Name}}Lt
type {{$each.Name}}Lt {{$each.Type}}

// Cond Cond
func (n {{$each.Name}}Lt) Cond() string {
	return "{{$each.Column}}{{$.Lt|raw}}?"
}

// Args Args
func (n {{$each.Name}}Lt) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n {{$each.Name}}Lt) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}Lt) Or(ors ...Filter) JoinableFilter {
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

// {{$each.Name}}BE {{$each.Name}}BE
type {{$each.Name}}BE {{$each.Type}}

// Cond Cond
func (n {{$each.Name}}BE) Cond() string {
	return "{{$each.Column}}{{$.Bt|raw}}=?"
}

// Args Args
func (n {{$each.Name}}BE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n {{$each.Name}}BE) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}BE) Or(ors ...Filter) JoinableFilter {
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

// {{$each.Name}}LE {{$each.Name}}LE
type {{$each.Name}}LE {{$each.Type}}

// Cond Cond
func (n {{$each.Name}}LE) Cond() string {
	return "{{$each.Column}}{{$.Lt|raw}}=?"
}

// Args Args
func (n {{$each.Name}}LE) Args() []interface{} {
	return []interface{}{n}
}

// And And
func (n {{$each.Name}}LE) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}LE) Or(ors ...Filter) JoinableFilter {
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

// {{$each.Name}}In {{$each.Name}}In
type {{$each.Name}}In []{{$each.Type}}

// Cond Cond
func (n {{$each.Name}}In) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("{{$each.Column}} in (%s)",strings.Join(placeHolders,","))
}

// Args Args
func (n {{$each.Name}}In) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n {{$each.Name}}In) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}In) Or(ors ...Filter) JoinableFilter {
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

// {{$each.Name}}NotIn {{$each.Name}}NotIn
type {{$each.Name}}NotIn []{{$each.Type}}

// Cond Cond
func (n {{$each.Name}}NotIn) Cond() string {
	placeHolders := make([]string, 0, len(n))
	for range n {
		placeHolders = append(placeHolders, "?")
	}
	return fmt.Sprintf("{{$each.Column}} not in (%s)",strings.Join(placeHolders,","))
}

// Args Args
func (n {{$each.Name}}NotIn) Args() []interface{} {
	args := make([]interface{}, 0, len(n))
	for _, each := range n {
		args = append(args, each)
	}
	return args
}

// And And
func (n {{$each.Name}}NotIn) And(ands ...Filter) JoinableFilter {
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
func (n {{$each.Name}}NotIn) Or(ors ...Filter) JoinableFilter {
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

// SortBy{{$each.Name}} SortBy{{$each.Name}}
func SortBy{{$each.Name}}(asc bool) JoinableSorterBuilder {
	if asc {
		return Sorter("{{$each.Column}} asc")
	}
	return Sorter("{{$each.Column}} desc")
}

{{end}}
`
