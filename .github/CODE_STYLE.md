# Code style

🚧 work in progress! 🚧

## Add `String() string` methods

Add `String() string` methods to new types, so they can easily be logged.
💁 You should try de-referencing pointer fields in your method, to avoid logging pointer addresses.

## Subtests with mocks

We use `gomock` to use `mockgen`-generated mocks.

This is trivial to use with single test, but it can get tedious to use with subtests.

In the following we use this example production code:

```go
//go:generate mockgen -destination=mock_multiplier_test.go -package $GOPACKAGE . Multiplier

type Multiplier interface {
  Multiply(n int, by int) (result int)
}

// Function we want to test
func multiplyByTwo(n int, multiplier Multiplier) (result int) {
  return multiplier.Multiply(n, 2)
}
```

In your tests, since you need to define a controller

```go
ctrl := gomock.NewController(t)
```

before configuring your mocks, it means you must **create the controller and configure your mocks in your subtest** and not in the parent test. Otherwise a subtest could crash the parent test and failure logs will look strange.

⛔ this is **bad**:

```go
func Test_multiplyByTwo(t *testing.T) {
    ctrl := gomock.NewController(t)

    multiplier3 := NewMockMultiplier(ctrl)
    multiplier3.EXPECT().
        Multiply(3, 2).Return(6)

    testCases := map[string]struct {
        n          int
        multiplier Multiplier
        result     int
    }{
        "3 by 2": {
            n:          3,
            multiplier: multiplier3,
            result:     6,
        },
    }

    for name, testCase := range testCases {
        t.Run(name, func(t *testing.T) {
            result := multiplyByTwo(testCase.n, testCase.multiplier)

            assert.Equal(t, testCase.result, result)
        })
    }
}
```

By default, you should aim to:

1. Specify the mock(s) expected arguments and returns in your test cases slice/map
1. Configure the mock(s) in your subtest

Corresponding example test:

```go
func Test_multiplyByTwo(t *testing.T) {
    testCases := map[string]struct {
        n               int
        multiplierBy    int
        multiplerResult int
        result          int
    }{
        "3 by 2": {
            n:               3,
            multiplierBy:    2,
            multiplerResult: 6,
            result:          6,
        },
    }

    for name, testCase := range testCases {
        t.Run(name, func(t *testing.T) {
            ctrl := gomock.NewController(t)

            multiplier := NewMockMultiplier(ctrl)
            multiplier.EXPECT().
                Multiply(testCase.n, testCase.multiplierBy).
                Return(testCase.multiplerResult)

            result := multiplyByTwo(testCase.n, multiplier)

            assert.Equal(t, testCase.result, result)
        })
    }
}
```

Now there is an exception where your mocks configuration change a lot from a test case to another. This is seen with **at least two levels** of `if` conditions nesting to configure your mocks. In this case, you shall avoid having a test cases structure (slice/map) and run each subtest independently. For example:

```go
func Test_(t *testing.T) {
    t.Run("case 1", func(t *testing.T) {
        ctrl := gomock.NewController(t)
        // ...
    })

    // ...

    t.Run("case n", func(t *testing.T) {
        ctrl := gomock.NewController(t)
        // ...
    })
}
```

💡 this is usually a code smell where the production function being tested is too long/complex.
So ideally try to refactor the production code first if you can.
