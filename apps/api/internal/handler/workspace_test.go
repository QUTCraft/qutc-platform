package handler

import "testing"

func validApplicationFixture() applicationRequest {
	return applicationRequest{
		Type:      "whitelist",
		ClassName: "计算机231",
		Name:      "Yukino",
		GameID:    "YukinoCraft",
		QQNumber:  "123456789",
		Email:     "yukino@example.com",
		Note:      "希望参与周末建筑测试。",
	}
}

func TestValidApplicationRequest(t *testing.T) {
	if !validApplicationRequest(validApplicationFixture()) {
		t.Fatal("expected fixture to be valid")
	}

	tests := []struct {
		name   string
		mutate func(*applicationRequest)
	}{
		{name: "invalid type", mutate: func(value *applicationRequest) { value.Type = "admin" }},
		{name: "invalid email", mutate: func(value *applicationRequest) { value.Email = "not-an-email" }},
		{name: "invalid qq number", mutate: func(value *applicationRequest) { value.QQNumber = "1234" }},
		{name: "missing name", mutate: func(value *applicationRequest) { value.Name = "  " }},
		{name: "long note", mutate: func(value *applicationRequest) { value.Note = string(make([]rune, 501)) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validApplicationFixture()
			test.mutate(&value)
			if validApplicationRequest(value) {
				t.Fatalf("expected %s to be rejected", test.name)
			}
		})
	}
}

func TestValidApplicationRequestDefaultsToWhitelist(t *testing.T) {
	value := validApplicationFixture()
	value.Type = ""
	if !validApplicationRequest(value) {
		t.Fatal("expected an omitted application type to default to whitelist")
	}
}
