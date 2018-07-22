package tools

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/mail"
	"strings"
)

/*
Copyright 2018 Nicolas JUHEL

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

type MailAddress struct {
	mail.Address
}

type ListMailAddress map[int]*MailAddress

func CleanMergeSlice(str []string, args ...string) []string {
	for _, s := range args {
		if s != "" {
			str = append(str, s)
		}
	}

	return str
}

func UnicSliceString(str []string) []string {
	var new = make([]string, 0)

	for _, s := range str {
		if !ExistSliceString(new, s) {
			new = append(new, s)
		}
	}

	return new
}

func ExistSliceString(slc []string, str string) bool {
	for _, s := range slc {
		if s == str {
			return true
		}
	}

	return false
}

func CleanJoin(str []string, glue string) string {
	var new = make([]string, 0)

	for _, s := range str {
		if s != "" {
			new = append(new, s)
		}
	}

	return strings.Join(new, glue)
}

func MailAddressParser(str string) *MailAddress {
	obj, err := mail.ParseAddress(str)

	if err != nil {
		obj = &mail.Address{
			Name: str,
		}
	}

	return &MailAddress{
		*obj,
	}
}

func NewMailAddress(name, address string) *MailAddress {
	return &MailAddress{
		mail.Address{
			Name:    name,
			Address: address,
		},
	}
}

func (adr MailAddress) String() string {
	return strings.TrimSpace(adr.Address.String())
}

func (adr MailAddress) AddressOnly() string {
	str := strings.TrimSpace(adr.Address.String())

	if adr.Address.Name == "" || adr.Address.Address == "" || strings.HasPrefix(str, "<") {
		str = strings.Replace(str, "\n", "", -1)
		str = strings.Trim(str, ">")
		str = strings.Trim(str, "@")
		str = strings.Trim(str, "<")
		str = strings.TrimSpace(str)
		str = strings.Trim(str, "\"")
	} else {
		str = strings.TrimSpace(adr.Address.Address)
		str = strings.Replace(str, "\n", "", -1)
	}

	return str
}

func NewListMailAddress() ListMailAddress {
	return make(ListMailAddress, 0)
}

func (lst ListMailAddress) Len() int {
	return len(lst)
}

func (lst ListMailAddress) IsEmpty() bool {
	return len(lst) < 1
}

func (lst ListMailAddress) Merge(list ListMailAddress) {
	if list == nil {
		return
	}

	for _, adr := range list {
		lst[lst.Len()] = adr
	}
}

func (lst ListMailAddress) Add(adr *MailAddress) {
	lst[lst.Len()] = adr
}

func (lst ListMailAddress) AddParseEmail(m string) {
	lst.Add(MailAddressParser(m))
}

func (lst ListMailAddress) AddNewEmail(name, addr string) {
	lst.Add(NewMailAddress(name, addr))
}

func (lst ListMailAddress) AddressOnly() string {
	var res = make([]string, 0)

	for _, m := range lst {
		adr := m.AddressOnly()
		if adr != "" {
			res = append(res, adr)
		}
	}

	return strings.Join(res, ",")
}

func (lst ListMailAddress) String() string {
	var res = make([]string, 0)

	for _, m := range lst {
		adr := m.String()
		if adr != "" {
			res = append(res, adr)
		}
	}

	return strings.Join(res, ",")
}

func GenerateBoundary() (string, error) {
	var buf [30]byte

	_, err := io.ReadFull(rand.Reader, buf[:])

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", buf[:]), nil
}
