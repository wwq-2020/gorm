package main

const tplStr = `
// {{.Name}}Repo {{.Name}}Repo
type {{.Name}}Repo struct{
	db *sql.DB
}

// New{{.Name}}Repo New{{.Name}}Repo
func New{{.Name}}Repo(db *sql.DB) *{{.Name}}Repo {
	return &{{.Name}}Repo{
		db: db,
	}
}

// Find Find
func (rp {{$.Name}}Repo) Find(ctx context.Context, filter Filter, opts ...Option) ([]*{{.Name}}, error){
	options:=&options{}
	for _,opt := range opts {
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
		sql :=  fmt.Sprintf("{{.FindSQL}}%s%s", sortStr, paginate)
		rows, err = rp.db.QueryContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("{{.FindSQL}} where %s%s%s", filter.Cond(), sortStr, paginate)
		rows, err = rp.db.QueryContext(ctx, sql, filter.Args()... )
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
func (rp {{$.Name}}Repo) FindOne(ctx context.Context, filter Filter, opts ...Option) (*{{.Name}}, error){
	options:=&options{}
	for _,opt := range opts {
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
		sql :=  fmt.Sprintf("{{.FindSQL}}%s%s", sortStr, paginate)
		row= rp.db.QueryRowContext(ctx, sql)
	} else {
		sql := fmt.Sprintf("{{.FindSQL}} where %s%s%s", filter.Cond(), sortStr, paginate)
		row= rp.db.QueryRowContext(ctx, sql, filter.Args()... )
	}
	result := &{{.Name}}{}
	if err := row.Scan({{.Scan|raw}}); err !=nil {
		return nil, err
	}
	return result, nil
}

// Delete Delete
func (rp {{$.Name}}Repo) Delete(ctx context.Context, filter Filter) (int64, error){
	var result sql.Result
	var err error
	if filter.Cond() == "" {
		result, err = rp.db.ExecContext(ctx, "{{.DeleteSQL}}")
	} else {
		sql:=fmt.Sprintf("{{.DeleteSQL}} where %s", filter.Cond())
		result, err = rp.db.ExecContext(ctx, sql, filter.Args()... )
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
func (rp {{$.Name}}Repo) Update(ctx context.Context, filter Filter, updaters ...Updater) (int64, error){
	var result sql.Result
	var err error
	updateStrs := make([]string, 0, len(updaters))
	updateArgs := make([]interface{}, 0, len(updaters))
	for _, updater := range updaters {
		updateStrs = append(updateStrs, updater.Set())
		updateArgs = append(updateArgs, updater.Arg())
	}
	if filter.Cond() == "" {
		sqlBaseStr := "{{.UpdateSQL}} %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs,","))
		result, err = rp.db.ExecContext(ctx, sqlStr, updateArgs...)
	} else {
		sqlBaseStr := "{{.UpdateSQL}} %s where %s"
		sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs,","), filter.Cond())
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
func (rp {{.Name}}Repo) Create(ctx context.Context,obj *{{.Name}}) (int64, error) {
	result, err := rp.db.ExecContext(ctx, "{{.CreateSQL}} ({{.PlaceHolder}})", {{.Value}})
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
func (rp {{.Name}}Repo) BatchCreate(ctx context.Context, objs []*{{.Name}}) error {
	sqlBaseStr := "{{.CreateSQL}} %s"
	sqlPlaceHolder := make([]string, 0, len(objs))
	sqlArgs := make([]interface{}, 0, len(objs)*{{.ColumnCount}})
	for _, obj := range objs {
		sqlPlaceHolder = append(sqlPlaceHolder, "({{.PlaceHolder}})")
		sqlArgs = append(sqlArgs, {{.Value}})
	}
	sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(sqlPlaceHolder, ","))
	if _,err := rp.db.ExecContext(ctx, sqlStr, sqlArgs...); err != nil {
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
	paginate *paginate
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
func (n {{$each.Name}}Eq) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}Eq) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
func (n {{$each.Name}}NE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}NE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
func (n {{$each.Name}}Bt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}Bt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
func (n {{$each.Name}}Lt) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}Lt) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
func (n {{$each.Name}}BE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}BE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
func (n {{$each.Name}}LE) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}LE) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
func (n {{$each.Name}}In) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}In) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
func (n {{$each.Name}}NotIn) And(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}NotIn) Or(f Filter) JoinableFilter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
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
