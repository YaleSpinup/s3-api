package duck

import (
	"reflect"
	"testing"
)

func TestDefaultDuck(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want *DotDuck
	}{
		{
			name: "test name",
			args: args{
				name: "test",
			},
			want: &DotDuck{
				Protocol: "s3",
				Provider: "iterate GmbH",
				Nickname: "Spinup - test",
				Hostname: "s3.amazonaws.com",
				Port:     "443",
				Path:     "/test",
				WebURL:   "s3://test/",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultDuck(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultDuck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDotDuck_Generate(t *testing.T) {
	type fields struct {
		Protocol string
		Provider string
		Nickname string
		Hostname string
		Port     string
		Path     string
		WebURL   string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		// TODO
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DotDuck{
				Protocol: tt.fields.Protocol,
				Provider: tt.fields.Provider,
				Nickname: tt.fields.Nickname,
				Hostname: tt.fields.Hostname,
				Port:     tt.fields.Port,
				Path:     tt.fields.Path,
				WebURL:   tt.fields.WebURL,
			}
			got, err := d.Generate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DotDuck.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DotDuck.Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}
