# Changelog

## [v1.10.0](https://github.com/rabbitmq/amqp091-go/tree/v1.10.0) (2024-05-08)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.9.0...v1.10.0)

**Implemented enhancements:**

- Undeprecate non-context publish functions [\#259](https://github.com/rabbitmq/amqp091-go/pull/259) ([Zerpet](https://github.com/Zerpet))
- Update Go directive [\#257](https://github.com/rabbitmq/amqp091-go/pull/257) ([Zerpet](https://github.com/Zerpet))

**Fixed bugs:**

- republishing on reconnect bug in the example [\#249](https://github.com/rabbitmq/amqp091-go/issues/249)
- Channel Notify Close not receive event when connection is closed by RMQ server. [\#241](https://github.com/rabbitmq/amqp091-go/issues/241)
- Inconsistent documentation [\#231](https://github.com/rabbitmq/amqp091-go/issues/231)
- Data race in the client example [\#72](https://github.com/rabbitmq/amqp091-go/issues/72)
- Fix string function of URI [\#258](https://github.com/rabbitmq/amqp091-go/pull/258) ([Zerpet](https://github.com/Zerpet))

**Closed issues:**

- Documentation needed \(`PublishWithContext` does not use context\) [\#195](https://github.com/rabbitmq/amqp091-go/issues/195)
- concurrent dispatch data race [\#226](https://github.com/rabbitmq/amqp091-go/issues/226)

**Merged pull requests:**

- Fix data race in example [\#260](https://github.com/rabbitmq/amqp091-go/pull/260) ([Zerpet](https://github.com/Zerpet))
- Address CodeQL warning [\#252](https://github.com/rabbitmq/amqp091-go/pull/252) ([lukebakken](https://github.com/lukebakken))
- Add support for additional AMQP URI query parameters [\#251](https://github.com/rabbitmq/amqp091-go/pull/251) ([vilius-g](https://github.com/vilius-g))
- Example fix [\#250](https://github.com/rabbitmq/amqp091-go/pull/250) ([Boris-Plato](https://github.com/Boris-Plato))
- Increasing the code coverage [\#248](https://github.com/rabbitmq/amqp091-go/pull/248) ([edercarloscosta](https://github.com/edercarloscosta))
- Use correct mutex to guard confirms.published [\#240](https://github.com/rabbitmq/amqp091-go/pull/240) ([hjr265](https://github.com/hjr265))
- Documenting Publishing.Expiration usage [\#232](https://github.com/rabbitmq/amqp091-go/pull/232) ([niksteff](https://github.com/niksteff))
- fix comment typo in example\_client\_test.go [\#228](https://github.com/rabbitmq/amqp091-go/pull/228) ([wisaTong](https://github.com/wisaTong))
- Bump go.uber.org/goleak from 1.2.1 to 1.3.0 [\#227](https://github.com/rabbitmq/amqp091-go/pull/227) ([dependabot[bot]](https://github.com/apps/dependabot))

## [v1.9.0](https://github.com/rabbitmq/amqp091-go/tree/v1.9.0) (2023-10-02)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.8.1...v1.9.0)

**Implemented enhancements:**

- Use of buffered delivery channels when prefetch\_count is not null [\#200](https://github.com/rabbitmq/amqp091-go/issues/200)

**Fixed bugs:**

- connection block when write connection reset by peer [\#222](https://github.com/rabbitmq/amqp091-go/issues/222)
- Test failure on 32bit architectures [\#202](https://github.com/rabbitmq/amqp091-go/issues/202)

**Closed issues:**

- Add a constant to set consumer timeout as queue argument [\#201](https://github.com/rabbitmq/amqp091-go/issues/201)
- Add a constant for CQ version [\#199](https://github.com/rabbitmq/amqp091-go/issues/199)
- Examples may need to be updated after \#140 [\#153](https://github.com/rabbitmq/amqp091-go/issues/153)

**Merged pull requests:**

- Update spec091.go [\#224](https://github.com/rabbitmq/amqp091-go/pull/224) ([pinkfish](https://github.com/pinkfish))
- Closes 222 [\#223](https://github.com/rabbitmq/amqp091-go/pull/223) ([yywing](https://github.com/yywing))
- Update write.go [\#221](https://github.com/rabbitmq/amqp091-go/pull/221) ([pinkfish](https://github.com/pinkfish))
- Bump versions [\#219](https://github.com/rabbitmq/amqp091-go/pull/219) ([lukebakken](https://github.com/lukebakken))
- remove extra word 'accept' from ExchangeDeclare description [\#217](https://github.com/rabbitmq/amqp091-go/pull/217) ([a-sabzian](https://github.com/a-sabzian))
- Misc Windows CI updates [\#216](https://github.com/rabbitmq/amqp091-go/pull/216) ([lukebakken](https://github.com/lukebakken))
- Stop using deprecated Publish function [\#207](https://github.com/rabbitmq/amqp091-go/pull/207) ([Zerpet](https://github.com/Zerpet))
- Constant for consumer timeout queue argument [\#206](https://github.com/rabbitmq/amqp091-go/pull/206) ([Zerpet](https://github.com/Zerpet))
- Add a constant for CQ v2 queue argument [\#205](https://github.com/rabbitmq/amqp091-go/pull/205) ([Zerpet](https://github.com/Zerpet))
- Fix example for 32-bit compatibility [\#204](https://github.com/rabbitmq/amqp091-go/pull/204) ([Zerpet](https://github.com/Zerpet))
- Fix to increase timeout milliseconds since it's too tight [\#203](https://github.com/rabbitmq/amqp091-go/pull/203) ([t2y](https://github.com/t2y))
- Add Channel.ConsumeWithContext to be able to cancel delivering [\#192](https://github.com/rabbitmq/amqp091-go/pull/192) ([t2y](https://github.com/t2y))

## [v1.8.1](https://github.com/rabbitmq/amqp091-go/tree/v1.8.1) (2023-05-04)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.8.0...v1.8.1)

**Fixed bugs:**

- Fixed incorrect version reported in client properties [52ce2efd03c53dcf77d5496977da46840e9abd24](https://github.com/rabbitmq/amqp091-go/commit/52ce2efd03c53dcf77d5496977da46840e9abd24)

**Merged pull requests:**

- Fix Example Client not reconnecting [\#186](https://github.com/rabbitmq/amqp091-go/pull/186) ([frankfil](https://github.com/frankfil))

## [v1.8.0](https://github.com/rabbitmq/amqp091-go/tree/v1.8.0) (2023-03-21)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.7.0...v1.8.0)

**Closed issues:**

- memory leak  [\#179](https://github.com/rabbitmq/amqp091-go/issues/179)
-  the publishWithContext interface will not return when it times out [\#178](https://github.com/rabbitmq/amqp091-go/issues/178)

**Merged pull requests:**

- Fix race condition on confirms [\#183](https://github.com/rabbitmq/amqp091-go/pull/183) ([calloway-jacob](https://github.com/calloway-jacob))
- Add a CloseDeadline function to Connection [\#181](https://github.com/rabbitmq/amqp091-go/pull/181) ([Zerpet](https://github.com/Zerpet))
- Fix memory leaks [\#180](https://github.com/rabbitmq/amqp091-go/pull/180) ([GXKe](https://github.com/GXKe))
- Bump go.uber.org/goleak from 1.2.0 to 1.2.1 [\#177](https://github.com/rabbitmq/amqp091-go/pull/177) ([dependabot[bot]](https://github.com/apps/dependabot))

## [v1.7.0](https://github.com/rabbitmq/amqp091-go/tree/v1.7.0) (2023-02-09)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.6.1...v1.7.0)

**Closed issues:**

- \#31 resurfacing \(?\) [\#170](https://github.com/rabbitmq/amqp091-go/issues/170)
- Deprecate QueueInspect [\#167](https://github.com/rabbitmq/amqp091-go/issues/167)
- v1.6.0 causing rabbit connection errors [\#160](https://github.com/rabbitmq/amqp091-go/issues/160)

**Merged pull requests:**

- Set channels and allocator to nil in shutdown [\#172](https://github.com/rabbitmq/amqp091-go/pull/172) ([lukebakken](https://github.com/lukebakken))
- Fix racing in Open [\#171](https://github.com/rabbitmq/amqp091-go/pull/171) ([Zerpet](https://github.com/Zerpet))
- adding go 1.20 to tests [\#169](https://github.com/rabbitmq/amqp091-go/pull/169) ([halilylm](https://github.com/halilylm))
- Deprecate the QueueInspect function [\#168](https://github.com/rabbitmq/amqp091-go/pull/168) ([lukebakken](https://github.com/lukebakken))
- Check if channel is nil before updating it [\#150](https://github.com/rabbitmq/amqp091-go/pull/150) ([julienschmidt](https://github.com/julienschmidt))

## [v1.6.1](https://github.com/rabbitmq/amqp091-go/tree/v1.6.1) (2023-02-01)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.6.1-rc.2...v1.6.1)

**Merged pull requests:**

- Update Makefile targets related to RabbitMQ [\#163](https://github.com/rabbitmq/amqp091-go/pull/163) ([Zerpet](https://github.com/Zerpet))

## [v1.6.1-rc.2](https://github.com/rabbitmq/amqp091-go/tree/v1.6.1-rc.2) (2023-01-31)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.6.1-rc.1...v1.6.1-rc.2)

**Merged pull requests:**

- Do not overly protect writes [\#162](https://github.com/rabbitmq/amqp091-go/pull/162) ([lukebakken](https://github.com/lukebakken))

## [v1.6.1-rc.1](https://github.com/rabbitmq/amqp091-go/tree/v1.6.1-rc.1) (2023-01-31)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.6.0...v1.6.1-rc.1)

**Closed issues:**

- Calling Channel\(\) on an empty connection panics [\#148](https://github.com/rabbitmq/amqp091-go/issues/148)

**Merged pull requests:**

- Ensure flush happens and correctly lock connection for a series of unflushed writes [\#161](https://github.com/rabbitmq/amqp091-go/pull/161) ([lukebakken](https://github.com/lukebakken))

## [v1.6.0](https://github.com/rabbitmq/amqp091-go/tree/v1.6.0) (2023-01-20)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.5.0...v1.6.0)

**Implemented enhancements:**

- Add constants for Queue arguments [\#145](https://github.com/rabbitmq/amqp091-go/pull/145) ([Zerpet](https://github.com/Zerpet))

**Closed issues:**

- README not up to date [\#154](https://github.com/rabbitmq/amqp091-go/issues/154)
- Allow re-using default connection config \(custom properties\) [\#152](https://github.com/rabbitmq/amqp091-go/issues/152)
- Rename package name to amqp in V2 [\#151](https://github.com/rabbitmq/amqp091-go/issues/151)
- Helper types to declare quorum queues [\#144](https://github.com/rabbitmq/amqp091-go/issues/144)
- Inefficient use of buffers reduces potential throughput for basicPublish with small messages. [\#141](https://github.com/rabbitmq/amqp091-go/issues/141)
- bug, close cause panic [\#130](https://github.com/rabbitmq/amqp091-go/issues/130)
- Publishing Headers are unable to store Table with slice values [\#125](https://github.com/rabbitmq/amqp091-go/issues/125)
- Example client can deadlock in Close due to unconsumed confirmations [\#122](https://github.com/rabbitmq/amqp091-go/issues/122)
- SAC not working properly [\#106](https://github.com/rabbitmq/amqp091-go/issues/106)

**Merged pull requests:**

- Add automatic CHANGELOG.md generation [\#158](https://github.com/rabbitmq/amqp091-go/pull/158) ([lukebakken](https://github.com/lukebakken))
- Supply library-defined props with NewConnectionProperties [\#157](https://github.com/rabbitmq/amqp091-go/pull/157) ([slagiewka](https://github.com/slagiewka))
- Fix linter warnings [\#156](https://github.com/rabbitmq/amqp091-go/pull/156) ([Zerpet](https://github.com/Zerpet))
- Remove outdated information from README [\#155](https://github.com/rabbitmq/amqp091-go/pull/155) ([scriptcoded](https://github.com/scriptcoded))
- Add example producer using DeferredConfirm [\#149](https://github.com/rabbitmq/amqp091-go/pull/149) ([Zerpet](https://github.com/Zerpet))
- Ensure code is formatted [\#147](https://github.com/rabbitmq/amqp091-go/pull/147) ([lukebakken](https://github.com/lukebakken))
- Fix inefficient use of buffers that reduces the potential throughput of basicPublish [\#142](https://github.com/rabbitmq/amqp091-go/pull/142) ([fadams](https://github.com/fadams))
- Do not embed context in DeferredConfirmation [\#140](https://github.com/rabbitmq/amqp091-go/pull/140) ([tie](https://github.com/tie))
- Add constant for default exchange [\#139](https://github.com/rabbitmq/amqp091-go/pull/139) ([marlongerson](https://github.com/marlongerson))
- Fix indentation and remove unnecessary instructions [\#138](https://github.com/rabbitmq/amqp091-go/pull/138) ([alraujo](https://github.com/alraujo))
- Remove unnecessary instruction [\#135](https://github.com/rabbitmq/amqp091-go/pull/135) ([alraujo](https://github.com/alraujo))
- Fix example client to avoid deadlock in Close [\#123](https://github.com/rabbitmq/amqp091-go/pull/123) ([Zerpet](https://github.com/Zerpet))
- Bump go.uber.org/goleak from 1.1.12 to 1.2.0 [\#116](https://github.com/rabbitmq/amqp091-go/pull/116) ([dependabot[bot]](https://github.com/apps/dependabot))

## [v1.5.0](https://github.com/rabbitmq/amqp091-go/tree/v1.5.0) (2022-09-07)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.4.0...v1.5.0)

**Implemented enhancements:**

- Provide a friendly way to set connection name [\#105](https://github.com/rabbitmq/amqp091-go/issues/105)

**Closed issues:**

- Support connection.update-secret  [\#107](https://github.com/rabbitmq/amqp091-go/issues/107)
- Example Client: Implementation of a Consumer with reconnection support [\#40](https://github.com/rabbitmq/amqp091-go/issues/40)

**Merged pull requests:**

- use PublishWithContext instead of Publish [\#115](https://github.com/rabbitmq/amqp091-go/pull/115) ([Gsantomaggio](https://github.com/Gsantomaggio))
- Add support for connection.update-secret [\#114](https://github.com/rabbitmq/amqp091-go/pull/114) ([Zerpet](https://github.com/Zerpet))
- Remove warning on RabbitMQ tutorials in go [\#113](https://github.com/rabbitmq/amqp091-go/pull/113) ([ChunyiLyu](https://github.com/ChunyiLyu))
- Update AMQP Spec [\#110](https://github.com/rabbitmq/amqp091-go/pull/110) ([Zerpet](https://github.com/Zerpet))
- Add an example of reliable consumer [\#109](https://github.com/rabbitmq/amqp091-go/pull/109) ([Zerpet](https://github.com/Zerpet))
- Add convenience function to set connection name [\#108](https://github.com/rabbitmq/amqp091-go/pull/108) ([Zerpet](https://github.com/Zerpet))

## [v1.4.0](https://github.com/rabbitmq/amqp091-go/tree/v1.4.0) (2022-07-19)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.3.4...v1.4.0)

**Closed issues:**

- target machine actively refused connection [\#99](https://github.com/rabbitmq/amqp091-go/issues/99)
- 504 channel/connection is not open error occurred in multiple connection with same rabbitmq service [\#97](https://github.com/rabbitmq/amqp091-go/issues/97)
- Add possible cancel of DeferredConfirmation [\#92](https://github.com/rabbitmq/amqp091-go/issues/92)
- Documentation  [\#89](https://github.com/rabbitmq/amqp091-go/issues/89)
- Channel Close gets stuck after closing a connection \(via management UI\) [\#88](https://github.com/rabbitmq/amqp091-go/issues/88)
- this library has same issue [\#83](https://github.com/rabbitmq/amqp091-go/issues/83)
- Provide a logging interface [\#81](https://github.com/rabbitmq/amqp091-go/issues/81)
- 1.4.0 release checklist [\#77](https://github.com/rabbitmq/amqp091-go/issues/77)
- Data race in the client example [\#72](https://github.com/rabbitmq/amqp091-go/issues/72)
- reader go routine hangs and leaks when Connection.Close\(\) is called multiple times [\#69](https://github.com/rabbitmq/amqp091-go/issues/69)
- Support auto-reconnect and cluster [\#65](https://github.com/rabbitmq/amqp091-go/issues/65)
- Connection/Channel Deadlock [\#32](https://github.com/rabbitmq/amqp091-go/issues/32)
- Closing connection and/or channel hangs NotifyPublish is used [\#21](https://github.com/rabbitmq/amqp091-go/issues/21)
- Consumer channel isn't closed in the event of unexpected disconnection [\#18](https://github.com/rabbitmq/amqp091-go/issues/18)

**Merged pull requests:**

- fix race condition with context close and confirm at the same time on DeferredConfirmation. [\#101](https://github.com/rabbitmq/amqp091-go/pull/101) ([sapk](https://github.com/sapk))
- Add build TLS config from URI [\#98](https://github.com/rabbitmq/amqp091-go/pull/98) ([reddec](https://github.com/reddec))
- Use context for Publish methods [\#96](https://github.com/rabbitmq/amqp091-go/pull/96) ([sapk](https://github.com/sapk))
- Added function to get the remote peer's IP address \(conn.RemoteAddr\(\)\) [\#95](https://github.com/rabbitmq/amqp091-go/pull/95) ([rabb1t](https://github.com/rabb1t))
- Update connection documentation [\#90](https://github.com/rabbitmq/amqp091-go/pull/90) ([Zerpet](https://github.com/Zerpet))
- Revert test to demonstrate actual bug [\#87](https://github.com/rabbitmq/amqp091-go/pull/87) ([lukebakken](https://github.com/lukebakken))
- Minor improvements to examples [\#86](https://github.com/rabbitmq/amqp091-go/pull/86) ([lukebakken](https://github.com/lukebakken))
- Do not skip flaky test in CI [\#85](https://github.com/rabbitmq/amqp091-go/pull/85) ([lukebakken](https://github.com/lukebakken))
- Add logging [\#84](https://github.com/rabbitmq/amqp091-go/pull/84) ([lukebakken](https://github.com/lukebakken))
- Add a win32 build [\#82](https://github.com/rabbitmq/amqp091-go/pull/82) ([lukebakken](https://github.com/lukebakken))
- channel: return nothing instead of always a nil-error in receive methods [\#80](https://github.com/rabbitmq/amqp091-go/pull/80) ([fho](https://github.com/fho))
- update the contributing & readme files, improve makefile [\#79](https://github.com/rabbitmq/amqp091-go/pull/79) ([fho](https://github.com/fho))
- Fix lint errors [\#78](https://github.com/rabbitmq/amqp091-go/pull/78) ([lukebakken](https://github.com/lukebakken))
- ci: run golangci-lint [\#76](https://github.com/rabbitmq/amqp091-go/pull/76) ([fho](https://github.com/fho))
- ci: run test via make & remove travis CI config [\#75](https://github.com/rabbitmq/amqp091-go/pull/75) ([fho](https://github.com/fho))
- ci: run tests with race detector [\#74](https://github.com/rabbitmq/amqp091-go/pull/74) ([fho](https://github.com/fho))
- Detect go routine leaks in integration testcases [\#73](https://github.com/rabbitmq/amqp091-go/pull/73) ([fho](https://github.com/fho))
- connection: fix: reader go-routine is leaked on connection close [\#70](https://github.com/rabbitmq/amqp091-go/pull/70) ([fho](https://github.com/fho))
- adding best practises for NotifyPublish for issue\_21 scenario [\#68](https://github.com/rabbitmq/amqp091-go/pull/68) ([DanielePalaia](https://github.com/DanielePalaia))
- Update Go version [\#67](https://github.com/rabbitmq/amqp091-go/pull/67) ([Zerpet](https://github.com/Zerpet))
- Regenerate certs with SHA256 to fix test with Go 1.18+ [\#66](https://github.com/rabbitmq/amqp091-go/pull/66) ([anthonyfok](https://github.com/anthonyfok))

## [v1.3.4](https://github.com/rabbitmq/amqp091-go/tree/v1.3.4) (2022-04-01)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.3.3...v1.3.4)

**Merged pull requests:**

- bump version to 1.3.4 [\#63](https://github.com/rabbitmq/amqp091-go/pull/63) ([DanielePalaia](https://github.com/DanielePalaia))
- updating doc [\#62](https://github.com/rabbitmq/amqp091-go/pull/62) ([DanielePalaia](https://github.com/DanielePalaia))

## [v1.3.3](https://github.com/rabbitmq/amqp091-go/tree/v1.3.3) (2022-04-01)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.3.2...v1.3.3)

**Closed issues:**

- Add Client Version [\#49](https://github.com/rabbitmq/amqp091-go/issues/49)
- OpenTelemetry Propagation [\#22](https://github.com/rabbitmq/amqp091-go/issues/22)

**Merged pull requests:**

- bump buildVersion for release [\#61](https://github.com/rabbitmq/amqp091-go/pull/61) ([DanielePalaia](https://github.com/DanielePalaia))
- adding documentation for notifyClose best pratices [\#60](https://github.com/rabbitmq/amqp091-go/pull/60) ([DanielePalaia](https://github.com/DanielePalaia))
- adding documentation on NotifyClose of connection and channel to enfo… [\#59](https://github.com/rabbitmq/amqp091-go/pull/59) ([DanielePalaia](https://github.com/DanielePalaia))

## [v1.3.2](https://github.com/rabbitmq/amqp091-go/tree/v1.3.2) (2022-03-28)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.3.1...v1.3.2)

**Closed issues:**

- Potential race condition in Connection module [\#31](https://github.com/rabbitmq/amqp091-go/issues/31)

**Merged pull requests:**

- bump versioning to 1.3.2 [\#58](https://github.com/rabbitmq/amqp091-go/pull/58) ([DanielePalaia](https://github.com/DanielePalaia))

## [v1.3.1](https://github.com/rabbitmq/amqp091-go/tree/v1.3.1) (2022-03-25)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.3.0...v1.3.1)

**Closed issues:**

- Possible deadlock on DeferredConfirmation.Wait\(\) [\#46](https://github.com/rabbitmq/amqp091-go/issues/46)
- Call to Delivery.Ack blocks indefinitely in case of disconnection [\#19](https://github.com/rabbitmq/amqp091-go/issues/19)
- Unexpacted behavor of channel.IsClosed\(\) [\#14](https://github.com/rabbitmq/amqp091-go/issues/14)
- A possible dead lock in connection close notification Go channel [\#11](https://github.com/rabbitmq/amqp091-go/issues/11)

**Merged pull requests:**

- These ones were the ones testing Open scenarios. The issue is that Op… [\#57](https://github.com/rabbitmq/amqp091-go/pull/57) ([DanielePalaia](https://github.com/DanielePalaia))
- changing defaultVersion to buildVersion and create a simple change\_ve… [\#54](https://github.com/rabbitmq/amqp091-go/pull/54) ([DanielePalaia](https://github.com/DanielePalaia))
- adding integration test for issue 11 [\#50](https://github.com/rabbitmq/amqp091-go/pull/50) ([DanielePalaia](https://github.com/DanielePalaia))
- Remove the old link product [\#48](https://github.com/rabbitmq/amqp091-go/pull/48) ([Gsantomaggio](https://github.com/Gsantomaggio))
- Fix deadlock on DeferredConfirmations [\#47](https://github.com/rabbitmq/amqp091-go/pull/47) ([SpencerTorres](https://github.com/SpencerTorres))
- Example client: Rename Stream\(\) to Consume\(\)  to avoid confusion with RabbitMQ streams [\#39](https://github.com/rabbitmq/amqp091-go/pull/39) ([andygrunwald](https://github.com/andygrunwald))
- Example client: Rename `name` to `queueName` to make the usage clear and explicit [\#38](https://github.com/rabbitmq/amqp091-go/pull/38) ([andygrunwald](https://github.com/andygrunwald))
- Client example: Renamed concept "Session" to "Client" [\#37](https://github.com/rabbitmq/amqp091-go/pull/37) ([andygrunwald](https://github.com/andygrunwald))
- delete unuseful code [\#36](https://github.com/rabbitmq/amqp091-go/pull/36) ([liutaot](https://github.com/liutaot))
- Client Example: Fix closing order [\#35](https://github.com/rabbitmq/amqp091-go/pull/35) ([andygrunwald](https://github.com/andygrunwald))
- Client example: Use instance logger instead of global logger [\#34](https://github.com/rabbitmq/amqp091-go/pull/34) ([andygrunwald](https://github.com/andygrunwald))

## [v1.3.0](https://github.com/rabbitmq/amqp091-go/tree/v1.3.0) (2022-01-13)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.2.0...v1.3.0)

**Closed issues:**

- documentation of changes triggering version updates [\#29](https://github.com/rabbitmq/amqp091-go/issues/29)
- Persistent messages folder [\#27](https://github.com/rabbitmq/amqp091-go/issues/27)

**Merged pull requests:**

- Expose a method to enable out-of-order Publisher Confirms [\#33](https://github.com/rabbitmq/amqp091-go/pull/33) ([benmoss](https://github.com/benmoss))
- Fix Signed 8-bit headers being treated as unsigned [\#26](https://github.com/rabbitmq/amqp091-go/pull/26) ([alex-goodisman](https://github.com/alex-goodisman))

## [v1.2.0](https://github.com/rabbitmq/amqp091-go/tree/v1.2.0) (2021-11-17)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/v1.1.0...v1.2.0)

**Closed issues:**

- No access to this vhost [\#24](https://github.com/rabbitmq/amqp091-go/issues/24)
- copyright issue? [\#12](https://github.com/rabbitmq/amqp091-go/issues/12)
- A possible dead lock when publishing message with confirmation  [\#10](https://github.com/rabbitmq/amqp091-go/issues/10)
- Semver release [\#7](https://github.com/rabbitmq/amqp091-go/issues/7)

**Merged pull requests:**

- Fix deadlock between publishing and receiving confirms [\#25](https://github.com/rabbitmq/amqp091-go/pull/25) ([benmoss](https://github.com/benmoss))
- Add GetNextPublishSeqNo for channel in confirm mode [\#23](https://github.com/rabbitmq/amqp091-go/pull/23) ([kamal-github](https://github.com/kamal-github))
- Added support for cert-only login without user and password [\#20](https://github.com/rabbitmq/amqp091-go/pull/20) ([mihaitodor](https://github.com/mihaitodor))

## [v1.1.0](https://github.com/rabbitmq/amqp091-go/tree/v1.1.0) (2021-09-21)

[Full Changelog](https://github.com/rabbitmq/amqp091-go/compare/ebd83429aa8cb06fa569473f623e87675f96d3a9...v1.1.0)

**Closed issues:**

- AMQPLAIN authentication does not work [\#15](https://github.com/rabbitmq/amqp091-go/issues/15)

**Merged pull requests:**

-  Fix AMQPLAIN authentication mechanism [\#16](https://github.com/rabbitmq/amqp091-go/pull/16) ([hodbn](https://github.com/hodbn))
- connection: clarify documented behavior of NotifyClose [\#13](https://github.com/rabbitmq/amqp091-go/pull/13) ([pabigot](https://github.com/pabigot))
- Add a link to pkg.go.dev API docs [\#9](https://github.com/rabbitmq/amqp091-go/pull/9) ([benmoss](https://github.com/benmoss))
- add test go version 1.16.x and 1.17.x [\#8](https://github.com/rabbitmq/amqp091-go/pull/8) ([k4n4ry](https://github.com/k4n4ry))
- fix typos [\#6](https://github.com/rabbitmq/amqp091-go/pull/6) ([h44z](https://github.com/h44z))
- Heartbeat interval should be timeout/2 [\#5](https://github.com/rabbitmq/amqp091-go/pull/5) ([ifo20](https://github.com/ifo20))
- Exporting Channel State [\#4](https://github.com/rabbitmq/amqp091-go/pull/4) ([eibrunorodrigues](https://github.com/eibrunorodrigues))
- Add codeql analysis [\#3](https://github.com/rabbitmq/amqp091-go/pull/3) ([MirahImage](https://github.com/MirahImage))
- Add PR github action. [\#2](https://github.com/rabbitmq/amqp091-go/pull/2) ([MirahImage](https://github.com/MirahImage))
- Update Copyright Statement [\#1](https://github.com/rabbitmq/amqp091-go/pull/1) ([rlewis24](https://github.com/rlewis24))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
