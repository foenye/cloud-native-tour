package main

import (
	"encoding/json"
)

type student struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentId string `json:"parentId"`
}

func studentLogic() {
	s := `{"id": 1, "name":"Derek", "parentId": "12321"}`
	var student student
	_ = json.Unmarshal([]byte(s), &student)
	println(student.ID, student.Name, student.ParentId)
	jsonBytes, _ := json.Marshal(student)
	println(string(jsonBytes))
}

func main() {
	studentLogic()
}
