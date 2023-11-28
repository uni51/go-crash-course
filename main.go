package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func initDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func validateUser(name string, age int) error {
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is empty")
	}
	if len(name) > 100 {
		return echo.NewHTTPError(http.StatusBadRequest, "name is too long")
	}
	if age < 0 || age >= 200 {
		return echo.NewHTTPError(http.StatusBadRequest, "age must be between 0 and 200")
	}
	return nil
}

func main() {
	db := initDB("example.db")
	e := echo.New()
	e.Use(middleware.Logger())

	// DELETEメソッドハンドラ：指定されたIDのユーザーを削除します。
	e.DELETE("/users/:id", func(c echo.Context) error {
		// リクエストパラメータからユーザーIDを取得します。
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			// IDを整数に変換できない場合、内部サーバーエラーを返します。
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// 指定されたIDのユーザーをデータベースから削除するDELETEクエリを実行します。
		result, err := db.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			// データベース操作中にエラーが発生した場合、内部サーバーエラーを返します。
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// データベースで影響を受けた行の数を確認します。
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// 影響を受けた行がない場合、指定されたIDのユーザーが見つかりませんでした。
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
		}

		// 操作が成功し、少なくとも1行が影響を受けた場合、成功応答とコンテンツなしを返します。
		return c.NoContent(http.StatusNoContent)
	})

	// "/users"へのPOSTリクエストに対するハンドラ
	e.POST("/users", func(c echo.Context) error {
		// フォームからユーザーの名前を取得
		name := c.FormValue("name")

		// フォームからユーザーの年齢を取得し、整数に変換
		age, _ := strconv.Atoi(c.FormValue("age"))

		// データベースに新しいユーザー情報を挿入するクエリを実行
		result, err := db.Exec("INSERT INTO users(name, age) VALUES(?, ?)", name, age)
		if err != nil {
			// エラーが発生した場合はInternal Server Errorを返す
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// 挿入された行のIDを取得
		id, _ := result.LastInsertId()

		// 挿入されたユーザー情報をJSON形式でクライアントに返す
		return c.JSON(http.StatusOK, &User{ID: int(id), Name: name, Age: age})
	})

	// "/users/:id"へのPUTリクエストに対するハンドラ
	e.PUT("/users/:id", func(c echo.Context) error {
		// パスパラメータからユーザーIDを取得し、整数に変換
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			// エラーが発生した場合はInternal Server Errorを返す
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// フォームからユーザーの名前を取得
		name := c.FormValue("name")

		// フォームからユーザーの年齢を取得し、整数に変換
		age, err := strconv.Atoi(c.FormValue("age"))
		if err != nil {
			// エラーが発生した場合はInternal Server Errorを返す
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// バリデーションの実行
		if err := validateUser(name, age); err != nil {
			return err
		}

		// データベースで指定されたユーザーIDの情報を更新するクエリを実行
		result, err := db.Exec("UPDATE users SET name = ?, age = ? WHERE id = ?", name, age, id)
		if err != nil {
			// エラーが発生した場合はInternal Server Errorを返す
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// 更新された行数を取得
		rows, _ := result.RowsAffected()
		// 更新された行数が0の場合はNot Foundを返す
		if rows == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
		}

		// 更新されたユーザー情報をJSON形式でクライアントに返す
		return c.JSON(http.StatusOK, &User{ID: id, Name: name, Age: age})
	})

	// "/users"へのGETリクエストに対するハンドラ
	e.GET("/users", func(c echo.Context) error {
		// データベースからユーザー情報を取得するクエリ
		rows, err := db.Query("SELECT id, name, age FROM users")
		if err != nil {
			// エラーが発生した場合はInternal Server Errorを返す
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		// 関数が終了する際に行をクローズする
		defer rows.Close()

		// ユーザー情報を格納するスライス
		users := []User{}
		// 取得した行を1行ずつ処理
		for rows.Next() {
			// User構造体の変数を宣言
			var user User
			// 行からデータをスキャンしてUser構造体に格納
			if err := rows.Scan(&user.ID, &user.Name, &user.Age); err != nil {
				// エラーが発生した場合はInternal Server Errorを返す
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			// ユーザーをスライスに追加
			users = append(users, user)
		}
		// 取得したユーザー情報をJSON形式でクライアントに返す
		return c.JSON(http.StatusOK, users)
	})

	// GETメソッドハンドラ：指定されたIDのユーザー情報を取得します。
	e.GET("/users/:id", func(c echo.Context) error {
		// リクエストパラメータからユーザーIDを取得します。
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			// IDを整数に変換できない場合、内部サーバーエラーを返します。
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// 指定されたIDのユーザー情報をデータベースから取得するSELECTクエリを実行します。
		row := db.QueryRow("SELECT id, name, age FROM users WHERE id = ?", id)

		// ユーザー情報を格納するための構造体を宣言します。
		var user User

		// クエリの結果をユーザー構造体にスキャンします。
		if err := row.Scan(&user.ID, &user.Name, &user.Age); err != nil {
			// エラーが発生した場合はInternal Server Errorを返します。
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// 取得したユーザー情報をJSON形式でクライアントに返します。
		return c.JSON(http.StatusOK, user)
	})

	e.Start(":8080")

	// db, err := sql.Open("sqlite3", "./example.db")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()

	// createTableSQL := `CREATE TABLE IF NOT EXISTS users (
	// 	id INTEGER PRIMARY KEY AUTOINCREMENT,
	// 	name TEXT NOT NULL,
	// 	age INTEGER NOT NULL
	// );
	// `

	// _, err = db.Exec(createTableSQL)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Println("Table created")
}
