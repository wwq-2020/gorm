# gorm

db, _= sql.Open("mysql","")
rp := testdata.New(db)
filter:= testdata.NameEq("test")
rp.Find(context.Backgroun(), filter)
