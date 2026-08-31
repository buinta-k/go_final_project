package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"project/pkg/db"
	"strings"
	"time"
)

type Api struct {
	DB *db.Base
}

func NewApi(Base *db.Base) *Api {
	Api := &Api{
		DB: Base,
	}

	return Api
}

type TasksResp struct {
	Tasks []*db.Task `json:"tasks"`
}

type password struct {
	Password string `json:"password"`
}

func (s *Api) NextDayHandler(res http.ResponseWriter, req *http.Request) {
	now := req.URL.Query().Get("now")
	dstart := req.URL.Query().Get("date")
	repeat := req.URL.Query().Get("repeat")

	Now, err := time.Parse("20060102", now)
	if err != nil {
		writeError(res, "Ошибка", http.StatusBadRequest)
		return
	}

	result, err := NextDate(Now, dstart, repeat)
	if err != nil {
		writeError(res, "Ошибка", http.StatusBadRequest)
		return
	}

	res.Write([]byte(result))

}

func (s *Api) TaskHandler(res http.ResponseWriter, req *http.Request) {
	var task db.Task
	err := json.NewDecoder(req.Body).Decode(&task)
	if err != nil {
		writeError(res, "Ошибка парсинга", http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		writeError(res, "Не указан заголовок", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if task.Date == "" {
		task.Date = now.Format("20060102")
	}

	_, err = time.Parse("20060102", task.Date)
	if err != nil {
		writeError(res, "Ошибка парсинга", http.StatusBadRequest)
		return
	}

	if task.Repeat != "" {
		repeatParts := strings.Split(task.Repeat, " ")
		_, err = Validate(repeatParts)
		if err != nil {
			writeError(res, "Ошибка парсинга", http.StatusBadRequest)
			return
		}
	}

	if task.Repeat == "" && task.Date < now.Format("20060102") {
		task.Date = now.Format("20060102")
	}

	var result string

	if task.Repeat != "" && task.Date < now.Format("20060102") && task.Date != now.Format("20060102") {
		result, err = NextDate(now, task.Date, task.Repeat)
		if err != nil {
			writeError(res, "Ошибка next date", http.StatusBadRequest)
			return
		}
		task.Date = result
	}

	id, err := s.DB.AddTask(&task)
	if err != nil {
		writeError(res, "Ошибка добавления в базу", http.StatusBadRequest)
		return
	}
	writeJson(res, db.Task{
		ID: fmt.Sprint(id),
	})
}

func (s *Api) TasksHandler(res http.ResponseWriter, req *http.Request) {
	searchParam := req.URL.Query().Get("search")

	var tasks []*db.Task
	var err error

	if searchParam == "" {
		tasks, err = s.DB.Tasks(50)
	} else {
		date, parseErr := time.Parse("02.01.2006", searchParam)

		if parseErr == nil {
			tasks, err = s.DB.SearchTasksDate(date.Format("20060102"), 50)
		} else {
			tasks, err = s.DB.SearchTasksParam(searchParam, 50)
		}
	}

	if err != nil {
		writeError(res, "Ошибка извлечения тасков", http.StatusBadRequest)
		return
	}

	writeJson(res, TasksResp{
		Tasks: tasks,
	})

}

func (s *Api) GetHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	task, err := s.DB.GetTask(id)
	if err != nil {
		writeError(res, "Задача не найдена", http.StatusBadRequest)
		return
	}

	writeJson(res, db.Task{
		ID:      task.ID,
		Date:    task.Date,
		Title:   task.Title,
		Comment: task.Comment,
		Repeat:  task.Repeat,
	})
}

func (s *Api) PutHandler(res http.ResponseWriter, req *http.Request) {
	var task db.Task

	err := json.NewDecoder(req.Body).Decode(&task)
	if err != nil {
		writeError(res, "Задача не найдена", http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		writeError(res, "Не указан заголовок", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if task.Date == "" {
		task.Date = now.Format("20060102")
	}

	_, err = time.Parse("20060102", task.Date)
	if err != nil {
		writeError(res, "Ошибка парсинга", http.StatusBadRequest)
		return
	}

	if task.Repeat != "" {
		repeatParts := strings.Split(task.Repeat, " ")
		_, err = Validate(repeatParts)
		if err != nil {
			writeError(res, "Ошибка парсинга", http.StatusBadRequest)
			return
		}
	}

	if task.Repeat == "" && task.Date < now.Format("20060102") {
		task.Date = now.Format("20060102")
	}

	var result string

	if task.Repeat != "" && task.Date < now.Format("20060102") && task.Date != now.Format("20060102") {
		result, err = NextDate(now, task.Date, task.Repeat)
		if err != nil {
			writeError(res, "Ошибка next date", http.StatusBadRequest)
			return
		}
		task.Date = result
	}

	err = s.DB.PutTask(task)
	if err != nil {
		writeError(res, "Задача не найдена", http.StatusBadRequest)
		return
	}

	writeJson(res, map[string]string{})
}

func (s *Api) ReadyHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	task, err := s.DB.GetTask(id)
	if err != nil {
		writeError(res, "Задача не найдена", http.StatusNotFound)
		return
	}

	if task.Repeat == "" {
		err = s.DB.DeleteTask(id)
		if err != nil {
			writeError(res, "Ошибка удаления", http.StatusBadRequest)
			return
		}
		writeJson(res, map[string]string{})
		return
	}

	date, err := NextDate(time.Now(), task.Date, task.Repeat)
	if err != nil {
		writeError(res, "Ошибка правила повторения", http.StatusBadRequest)
		return
	}

	err = s.DB.UpdateDate(date, id)
	if err != nil {
		writeError(res, "Ошибка обновления даты", http.StatusBadRequest)
		return
	}
	writeJson(res, map[string]string{})
}

func (s *Api) DeleteHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	err := s.DB.DeleteTask(id)

	if err != nil {
		writeError(res, "Ошибка удаления", http.StatusBadRequest)
		return
	}

	writeJson(res, map[string]string{})
}

func (s *Api) SignHandler(res http.ResponseWriter, req *http.Request) {
	var pass password

	err := json.NewDecoder(req.Body).Decode(&pass)
	if err != nil {
		writeError(res, "Ошибка парсинга пароля", http.StatusBadRequest)
		return
	}

	password := os.Getenv("TODO_PASSWORD")

	if pass.Password != password {
		writeError(res, "Неверный пароль", http.StatusUnauthorized)
		return
	}

	token, err := JWT(pass.Password)
	if err != nil {
		writeError(res, "Ошибка создания токена", http.StatusInternalServerError)
		return
	}

	writeJson(res, map[string]string{
		"token": token,
	})
}

func (s *Api) Init(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/nextdate", s.NextDayHandler)
	mux.HandleFunc("POST /api/task", auth(s.TaskHandler))
	mux.HandleFunc("GET /api/tasks", auth(s.TasksHandler))
	mux.HandleFunc("GET /api/task", auth(s.GetHandler))
	mux.HandleFunc("PUT /api/task", auth(s.PutHandler))
	mux.HandleFunc("POST /api/task/done", auth(s.ReadyHandler))
	mux.HandleFunc("DELETE /api/task", auth(s.DeleteHandler))
	mux.HandleFunc("POST /api/signin", s.SignHandler)
}
