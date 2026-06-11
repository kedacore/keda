# Changelog

All notable changes to this project will be documented in this file.

## [v0.12.0](https://github.com/paulmach/orb/compare/v0.11.1...v0.12.0) - 2025-09-17

### Fixed

-   Fix typos by [@NathanBaulch](https://github.com/NathanBaulch) in https://github.com/paulmach/orb/pull/157
-   Fix panic on reverse of empty linestrings by [@jo-me](https://github.com/jo-me) in https://github.com/paulmach/orb/pull/163
-   fix: return precisely 0.0 from mercator.ToGeo on arm64 by [@davidjb](https://github.com/davidjb) in https://github.com/paulmach/orb/pull/165

### Added

-   geojson: handle extra/foreign members in feature by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/164
-   encoding/mvt: Support for Marshalling to Proto Objects by [@kevinkreiser](https://github.com/kevinkreiser) in https://github.com/paulmach/orb/pull/154

## [v0.11.1](https://github.com/paulmach/orb/compare/v0.11.0...v0.11.1) - 2024-01-29

### Fixed

-   geojson: `null` json into non-pointer Feature/FeatureCollection will set them to empty by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/145

## [v0.11.0](https://github.com/paulmach/orb/compare/v0.10.0...v0.11.0) - 2024-01-11

### Fixed

-   quadtree: InBoundMatching does not properly accept passed-in buffer by [@nirmal-vuppuluri](https://github.com/nirmal-vuppuluri) in https://github.com/paulmach/orb/pull/139
-   mvt: Do not swallow error cause by [@m-pavel](https://github.com/m-pavel) in https://github.com/paulmach/orb/pull/137

### Changed

-   simplify: Visvalingam, by default, keeps 3 points for "areas" by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/140
-   encoding/mvt: skip encoding of features will nil geometry by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/141
-   encoding/wkt: improve unmarshalling performance by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/142

## [v0.10.0](https://github.com/paulmach/orb/compare/v0.9.2...v0.10.0) - 2023-07-16

### Added

-   add ChildrenInZoomRange method to maptile.Tile by [@peitili](https://github.com/peitili) in https://github.com/paulmach/orb/pull/133

## [v0.9.2](https://github.com/paulmach/orb/compare/v0.9.1...v0.9.2) - 2023-05-04

### Fixed

-   encoding/wkt: better handling/validation of missing parens by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/131

## [v0.9.1](https://github.com/paulmach/orb/compare/v0.9.0...v0.9.1) - 2023-04-26

### Fixed

-   Bump up mongo driver to 1.11.4 by [@m-pavel](https://github.com/m-pavel) in https://github.com/paulmach/orb/pull/129
-   encoding/wkt: split strings with regexp by [@m-pavel](https://github.com/m-pavel) in https://github.com/paulmach/orb/pull/128

## [v0.9.0](https://github.com/paulmach/orb/compare/v0.8.0...v0.9.0) - 2023-02-19

### Added

-   geojson: marshal/unmarshal BSON [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/123

## [v0.8.0](https://github.com/paulmach/orb/compare/v0.7.1...v0.8.0) - 2023-01-05

### Fixed

-   quadtree: fix bad sort due to pointer allocation issue by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/115
-   geojson: ensure geometry unmarshal errors get returned by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/117
-   encoding/mvt: remove use of crypto/md5 to compare marshalling in tests by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/118
-   encoding/wkt: fix panic for some invalid wkt data by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/119

### Other

-   fix typo by [@rubenpoppe](https://github.com/rubenpoppe) in https://github.com/paulmach/orb/pull/107
-   Fixed a small twister in README.md by [@Timahawk](https://github.com/Timahawk) in https://github.com/paulmach/orb/pull/108
-   update github ci to use go 1.19 by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/116

## [v0.7.1](https://github.com/paulmach/orb/compare/v0.7.0...v0.7.1) - 2022-05-16

No changes

The v0.7.0 tag was updated since it initially pointed to the wrong commit. This is causing caching issues.

## [v0.7.0](https://github.com/paulmach/orb/compare/v0.6.0...v0.7.0) - 2022-05-10

This tag is broken, please use v0.7.1 instead.

### Breaking Changes

-   tilecover now returns an error (vs. panicking) on non-closed 2d geometry by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/87

    This changes the signature of many of the methods in the [maptile/tilecover](https://github.com/paulmach/orb/tree/master/maptile/tilecover) package.
    To emulate the old behavior replace:

        tiles := tilecover.Geometry(poly, zoom)

    with

        tiles, err := tilecover.Geometry(poly, zoom)
        if err != nil {
        	panic(err)
        }

## [v0.6.0](https://github.com/paulmach/orb/compare/v0.5.0...v0.6.0) - 2022-05-04

### Added

-   geo: add correctly spelled LengthHaversine by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/97
-   geojson: add support for "external" json encoders/decoders by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/98
-   Add ewkb encoding/decoding support by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/88

## [v0.5.0](https://github.com/paulmach/orb/compare/v0.4.0...v0.5.0) - 2022-04-06

### Added

-   encoding/mvt: stable marshalling by [@travisgrigsby](https://github.com/travisgrigsby) in https://github.com/paulmach/orb/pull/93
-   encoding/mvt: support mvt marshal for GeometryCollection by [@dadadamarine](https://github.com/dadadamarine) in https://github.com/paulmach/orb/pull/89

### Fixed

-   quadtree: fix cleanup of nodes during removal by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/94

### Other

-   encoding/wkt: various code improvements by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/95
-   update protoscan to 0.2.1 by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/83

## [v0.4.0](https://github.com/paulmach/orb/compare/v0.3.0...v0.4.0) - 2021-11-11

### Added

-   geo: Add functions to calculate points based on distance and bearing by [@thzinc](https://github.com/thzinc) in https://github.com/paulmach/orb/pull/76

### Fixed

-   encoding/mvt: avoid reflect nil value by [@nicklasaven](https://github.com/nicklasaven) in https://github.com/paulmach/orb/pull/78

## [v0.3.0](https://github.com/paulmach/orb/compare/v0.2.2...v0.3.0) - 2021-10-16

### Changed

-   quadtree: sort KNearest results closest first by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/75
-   ring: require >=4 points to return true when calling Closed() by [@missinglink](https://github.com/missinglink) in https://github.com/paulmach/orb/pull/70

### Fixed

-   encoding/mvt: verify tile coord does not overflow for z > 20 by [@paulmach](https://github.com/paulmach) in https://github.com/paulmach/orb/pull/74
-   quadtree: Address panic-ing quadtree.Matching(…) method when finding no closest node by [@willsalz](https://github.com/willsalz) in https://github.com/paulmach/orb/pull/73

## [v0.2.2](https://github.com/paulmach/orb/compare/v0.2.1...v0.2.2) - 2021-06-05

### Fixed

-   Dependency resolution problems in some cases, issue https://github.com/paulmach/orb/issues/65, pr https://github.com/paulmach/orb/pull/66

## [v0.2.1](https://github.com/paulmach/orb/compare/v0.2.0...v0.2.1) - 2021-01-16

### Changed

-   encoding/mvt: upgrade protoscan v0.1 -> v0.2 [`ad31566`](https://github.com/paulmach/orb/commit/ad31566942027c1cd30dd341f35123fb54676599)
-   encoding/mvt: remove github.com/pkg/errors as a dependency [`d2e235`](https://github.com/paulmach/orb/commit/d2e23529a295a0d973cc787ad2742cb6ccbd5306)

## v0.2.0 - 2021-01-16

### Breaking Changes

-   Foreign Members in Feature Collections

    Extra attributes in a feature collection object will now be put into `featureCollection.ExtraMembers`.
    Similarly, stuff in `ExtraMembers will be marshalled into the feature collection base.
    The break happens if you were decoding these foreign members using something like

    ```go
    type MyFeatureCollection struct {
        geojson.FeatureCollection
        Title string `json:"title"`
    }
    ```

    **The above will no longer work** in this release and it never supported marshalling. See https://github.com/paulmach/orb/pull/56 for more details.

-   Features with nil/missing geometry will no longer return an errors

    Previously missing or invalid geometry in a feature collection would return a `ErrInvalidGeometry` error.
    However missing geometry is compliant with [section 3.2](https://tools.ietf.org/html/rfc7946#section-3.2) of the spec.
    See https://github.com/paulmach/orb/issues/38 and https://github.com/paulmach/orb/pull/58 for more details.

### Changed

-   encoding/mvt: faster unmarshalling for Mapbox Vector Tiles (MVT) see https://github.com/paulmach/orb/pull/57
