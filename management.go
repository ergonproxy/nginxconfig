package main

import "net/http"

type management struct{}

func (m management) ServerHTTP(w http.ResponseWriter, r *http.Request) {}
