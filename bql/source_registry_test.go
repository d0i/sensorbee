package bql

import (
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/sensorbee/sensorbee.v0/core"
	"testing"
)

func TestEmptyDefaultSourceCreatorRegistry(t *testing.T) {
	Convey("Given an empty default Source registry", t, func() {
		r := NewDefaultSourceCreatorRegistry()

		Convey("When adding a creator function", func() {
			err := r.Register("test_source", SourceCreatorFunc(createDummySource))

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When looking up a nonexistent creator", func() {
			_, err := r.Lookup("test_source")

			Convey("Then it should fail", func() {
				So(core.IsNotExist(err), ShouldBeTrue)
			})
		})

		Convey("When retrieving a list of creators", func() {
			m, err := r.List()

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)

				Convey("And the list should be empty", func() {
					So(m, ShouldBeEmpty)
				})
			})
		})

		Convey("When unregistering a nonexistent creator", func() {
			err := r.Unregister("test_source")

			Convey("Then it shouldn't fail", func() {
				So(core.IsNotExist(err), ShouldBeTrue)
			})
		})
	})
}

func TestDefaultSourceCreatorRegistry(t *testing.T) {
	ctx := core.NewContext(nil)

	Convey("Given an default Source registry having two types", t, func() {
		r := NewDefaultSourceCreatorRegistry()
		So(r.Register("TEST_source", SourceCreatorFunc(createDummySource)), ShouldBeNil)
		So(r.Register("TEST_source2", SourceCreatorFunc(createDummySource)), ShouldBeNil)

		Convey("When adding a new type having the registered type name", func() {
			err := r.Register("TEST_source", SourceCreatorFunc(createDummySource))

			Convey("Then it should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When looking up a creator", func() {
			c, err := r.Lookup("test_SOURCE2")

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)

				Convey("And it should have the expected type", func() {
					s, err := c.CreateSource(ctx, &IOParams{}, nil)
					So(err, ShouldBeNil)
					So(s, ShouldImplement, (*core.RewindableSource)(nil))
				})
			})
		})

		Convey("When retrieving a list of creators", func() {
			m, err := r.List()

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)

				Convey("And the list should have all creators", func() {
					So(len(m), ShouldEqual, 2)
					So(m["test_source"], ShouldNotBeNil)
					So(m["test_source2"], ShouldNotBeNil)
				})
			})
		})

		Convey("When unregistering a creator", func() {
			err := r.Unregister("test_SOURCE")

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)

				Convey("And the unregistered creator shouldn't be found", func() {
					_, err := r.Lookup("test_source")
					So(core.IsNotExist(err), ShouldBeTrue)
				})

				Convey("And the other creator should be found", func() {
					_, err := r.Lookup("test_source2")
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
