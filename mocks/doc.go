/*
Package mocks will have all the mocks of the library, we'll try to use mocking using blackbox
testing and integration tests whenever is possible.
*/
package mocks // import "github.com/spotahome/kooper/mocks"

// Operator tooling mocks.
//go:generate mockery -output ./controller -outpkg controller -dir ../controller -name Controller
//go:generate mockery -output ./controller -outpkg controller -dir ../controller -name Handler

// Wrappers mocks
//go:generate mockery -output ./wrapper/time -outpkg time -dir ../wrapper/time -name Time
