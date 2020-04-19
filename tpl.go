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
func (rp {{$.Name}}Repo) Find(ctx context.Context, filter Filter) ([]*{{.Name}}, error){
	sql:= fmt.Sprintf("{{.FindSQL}} where %s", filter.Cond())
	rows, err := rp.db.QueryContext(ctx, sql, filter.Args()... )
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

// Delete Delete
func (rp {{$.Name}}Repo) Delete(ctx context.Context, filter Filter) (int64, error){
	sql:=fmt.Sprintf("{{.DeleteSQL}} where %s", filter.Cond())
	result, err := rp.db.ExecContext(ctx, sql, filter.Args()... )
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
func (rp {{$.Name}}Repo) Update(ctx context.Context, filter Filter, updates ...Updater) (int64, error){
	updateStrs := make([]string, 0, len(updates))
	updateArgs := make([]interface{}, 0, len(updates))
	for _, update := range updates {
		updateStrs = append(updateStrs, update.Set())
		updateArgs = append(updateArgs, update.Arg())
	}
	sqlBaseStr := "{{.UpdateSQL}}  %s where %s"
	sqlStr := fmt.Sprintf(sqlBaseStr, strings.Join(updateStrs,","), filter.Cond())
	sqlArgs := append(updateArgs,filter.Args())
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

{{range $idx,$each := .Fields}}
// {{$each.Name}} {{$each.Name}}
type {{$each.Name}} {{$each.Type}}

// Cond Cond
func (n {{$each.Name}}) Cond() string {
	return "{{$each.Column}}=?"
}

// Args Args
func (n {{$each.Name}}) Args() []interface{} {
	return []interface{}{n}
}

// Set Set
func (n {{$each.Name}}) Set() string {
	return "{{$each.Column}}=?"
}

// Arg Arg
func (n {{$each.Name}}) Arg() interface{} {
	return n
}

// And And
func (n {{$each.Name}}) And(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") and (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}

// Or Or
func (n {{$each.Name}}) Or(f Filter) Filter {
	return &filter{
		cond: strings.Join([]string{"(", n.Cond(), ") or (", f.Cond(), ")"}, ""),
		args: append(n.Args(), f.Args()...),
	}
}
{{end}}
`
