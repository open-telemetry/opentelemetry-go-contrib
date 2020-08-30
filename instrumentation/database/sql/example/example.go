package main

import (
	"context"
	"database/sql"
	"fmt"
	"otelsql"

	"github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/label"
)

var (
	fooKey     = label.Key("ex.com/foo")
	barKey     = label.Key("ex.com/bar")
	anotherKey = label.Key("ex.com/another")
)

func main() {
	pusher, err := stdout.InstallNewPipeline([]stdout.Option{
		stdout.WithQuantiles([]float64{0.5, 0.9, 0.99}),
		stdout.WithPrettyPrint(),
	}, nil)
	if err != nil {
		panic(err)
	}
	defer pusher.Stop()

	opts := otelsql.WithTraceProvider(global.TraceProvider())
	otelsql.Register("otelsqlite3", &sqlite3.SQLiteDriver{}, opts)

	ctx := context.Background()
	ctx = correlation.NewContext(ctx,
		fooKey.String("foo1"),
		barKey.String("bar1"),
	)
	ctx = example(ctx)

	db, err := sql.Open("otelsqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sqlStmt := `
	create table bar (id integer not null primary key, name text);
	delete from bar;
	`
	_, err = db.ExecContext(ctx, sqlStmt)
	if err != nil {
		panic(err)
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		panic(err)
	}
	stmt, err := tx.PrepareContext(ctx, "insert into bar(id, name) values(?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	for i := 0; i < 6; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("otelsql-%v", i))
		if err != nil {
			panic(err)
		}
	}
	tx.Commit()

	rows, err := db.QueryContext(ctx, "select id, name from bar")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			panic(err)
		}
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	stmt, err = db.PrepareContext(ctx, "select name from bar where id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	var name string
	err = stmt.QueryRow("2").Scan(&name)
	if err != nil {
		panic(err)
	}

	_, err = db.ExecContext(ctx, "delete from bar")
	if err != nil {
		panic(err)
	}

	_, err = db.ExecContext(ctx, "insert into bar(id, name) values(1, 'foo'), (2, 'bar'), (3, 'baz')")
	if err != nil {
		panic(err)
	}
}

func example(ctx context.Context) context.Context {
	var span trace.Span
	ctx, span = global.Tracer("my-awesome-tracer-here").Start(ctx, "operation")
	defer span.End()

	span.AddEvent(ctx, "Nice operation!", label.Int("bogons", 100))
	span.SetAttributes(anotherKey.String("zebra"))
	return ctx
}
