package main

import "svinovoyna/internal/balance"
import "fmt"

func staticReport(c *balance.Config) { fmt.Println("static report:", c.Name) }
