(module
 (type $0 (func (param i32 i32) (result i32)))
 (type $1 (func (param i32) (result i32)))
 (type $2 (func (result i32)))
 (type $3 (func (param i32 i32 i32) (result i32)))
 (type $4 (func (param i32)))
 (type $5 (func))
 (type $6 (func (param i32 i32)))
 (type $7 (func (param i32 i32 i32)))
 (type $8 (func (param i32 i32 i32 i32)))
 (type $9 (func (param f64) (result i32)))
 (type $10 (func (param i32 i32 i32 i32) (result i32)))
 (type $11 (func (param i32 i32 f64 f64) (result i32)))
 (type $12 (func (param i32 i32 i64)))
 (type $13 (func (param i64 i64 i32 i64 i32) (result i32)))
 (type $14 (func (param f64)))
 (type $15 (func (param f64 f64) (result i32)))
 (type $16 (func (param i32) (result f64)))
 (import "env" "abort" (func $~lib/builtins/abort (param i32 i32 i32 i32)))
 (import "env" "console.log" (func $~lib/bindings/dom/console.log (param i32)))
 (global $assembly/index/agentId (mut i32) (i32.const 1056))
 (global $assembly/index/agentInitialized (mut i32) (i32.const 0))
 (global $assembly/index/modelType (mut i32) (i32.const 1056))
 (global $assembly/index/modelLoaded (mut i32) (i32.const 0))
 (global $assembly/index/externalInferenceEnabled (mut i32) (i32.const 0))
 (global $assembly/index/activeProvider (mut i32) (i32.const 1056))
 (global $~lib/rt/itcms/total (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/threshold (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/state (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/visitCount (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/pinSpace (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/iter (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/toSpace (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/white (mut i32) (i32.const 0))
 (global $~lib/rt/itcms/fromSpace (mut i32) (i32.const 0))
 (global $~lib/rt/tlsf/ROOT (mut i32) (i32.const 0))
 (global $assembly/index/apiKeys (mut i32) (i32.const 0))
 (global $assembly/index/providerEndpoints (mut i32) (i32.const 0))
 (global $assembly/index/providerModels (mut i32) (i32.const 0))
 (global $~lib/util/number/_frc_plus (mut i64) (i64.const 0))
 (global $~lib/util/number/_frc_minus (mut i64) (i64.const 0))
 (global $~lib/util/number/_exp (mut i32) (i32.const 0))
 (global $~lib/util/number/_K (mut i32) (i32.const 0))
 (global $~lib/util/number/_frc_pow (mut i64) (i64.const 0))
 (global $~lib/util/number/_exp_pow (mut i32) (i32.const 0))
 (global $~argumentsLength (mut i32) (i32.const 0))
 (global $~lib/rt/__rtti_base i32 (i32.const 13952))
 (global $~lib/memory/__stack_pointer (mut i32) (i32.const 46760))
 (memory $0 1)
 (data $0 (i32.const 1036) "\1c")
 (data $0.1 (i32.const 1048) "\02")
 (data $1 (i32.const 1068) "<")
 (data $1.1 (i32.const 1080) "\02\00\00\00(\00\00\00A\00l\00l\00o\00c\00a\00t\00i\00o\00n\00 \00t\00o\00o\00 \00l\00a\00r\00g\00e")
 (data $2 (i32.const 1132) "<")
 (data $2.1 (i32.const 1144) "\02\00\00\00 \00\00\00~\00l\00i\00b\00/\00r\00t\00/\00i\00t\00c\00m\00s\00.\00t\00s")
 (data $5 (i32.const 1260) "<")
 (data $5.1 (i32.const 1272) "\02\00\00\00$\00\00\00I\00n\00d\00e\00x\00 \00o\00u\00t\00 \00o\00f\00 \00r\00a\00n\00g\00e")
 (data $6 (i32.const 1324) ",")
 (data $6.1 (i32.const 1336) "\02\00\00\00\14\00\00\00~\00l\00i\00b\00/\00r\00t\00.\00t\00s")
 (data $8 (i32.const 1404) "<")
 (data $8.1 (i32.const 1416) "\02\00\00\00\1e\00\00\00~\00l\00i\00b\00/\00r\00t\00/\00t\00l\00s\00f\00.\00t\00s")
 (data $9 (i32.const 1468) ",")
 (data $9.1 (i32.const 1480) "\02\00\00\00\1c\00\00\00I\00n\00v\00a\00l\00i\00d\00 \00l\00e\00n\00g\00t\00h")
 (data $10 (i32.const 1516) "<")
 (data $10.1 (i32.const 1528) "\02\00\00\00&\00\00\00~\00l\00i\00b\00/\00a\00r\00r\00a\00y\00b\00u\00f\00f\00e\00r\00.\00t\00s")
 (data $11 (i32.const 1580) "L")
 (data $11.1 (i32.const 1592) "\02\00\00\000\00\00\00C\00r\00e\00a\00t\00i\00n\00g\00 \00n\00e\00w\00 \00A\00g\00e\00n\00t\00C\00o\00r\00e\00:\00 ")
 (data $12 (i32.const 1660) "L")
 (data $12.1 (i32.const 1672) "\02\00\00\000\00\00\00I\00n\00i\00t\00i\00a\00l\00i\00z\00i\00n\00g\00 \00A\00g\00e\00n\00t\00C\00o\00r\00e\00:\00 ")
 (data $13 (i32.const 1740) "\\")
 (data $13.1 (i32.const 1752) "\02\00\00\00D\00\00\00{\00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00A\00g\00e\00n\00t\00 \00n\00o\00t\00 \00i\00n\00i\00t\00i\00a\00l\00i\00z\00e\00d\00\"\00}")
 (data $14 (i32.const 1836) "L")
 (data $14.1 (i32.const 1848) "\02\00\00\008\00\00\00E\00x\00e\00c\00u\00t\00i\00n\00g\00 \00a\00g\00e\00n\00t\00 \00w\00i\00t\00h\00 \00i\00n\00p\00u\00t\00:\00 ")
 (data $15 (i32.const 1916) ",")
 (data $15.1 (i32.const 1928) "\02\00\00\00\16\00\00\00,\00 \00c\00o\00n\00t\00e\00x\00t\00:\00 ")
 (data $16 (i32.const 1964) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\10\00\00\00@\07\00\00\00\00\00\00\90\07")
 (data $17 (i32.const 2012) "l")
 (data $17.1 (i32.const 2024) "\02\00\00\00P\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00 \00\"\00r\00e\00s\00u\00l\00t\00\"\00:\00 \00\"\00P\00r\00o\00c\00e\00s\00s\00e\00d\00:\00 ")
 (data $18 (i32.const 2124) "<")
 (data $18.1 (i32.const 2136) "\02\00\00\00\1e\00\00\00\"\00,\00 \00\"\00a\00g\00e\00n\00t\00I\00d\00\"\00:\00 \00\"")
 (data $19 (i32.const 2188) "<")
 (data $19.1 (i32.const 2200) "\02\00\00\00\1e\00\00\00\"\00,\00 \00\"\00c\00o\00n\00t\00e\00x\00t\00\"\00:\00 \00\"")
 (data $20 (i32.const 2252) "\1c")
 (data $20.1 (i32.const 2264) "\02\00\00\00\04\00\00\00\"\00}")
 (data $21 (i32.const 2284) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\1c\00\00\00\f0\07\00\00\00\00\00\00`\08\00\00\00\00\00\00\a0\08\00\00\00\00\00\00\e0\08")
 (data $22 (i32.const 2332) "<")
 (data $22.1 (i32.const 2344) "\02\00\00\00 \00\00\00E\00x\00e\00c\00u\00t\00i\00n\00g\00 \00t\00o\00o\00l\00:\00 ")
 (data $23 (i32.const 2396) "<")
 (data $23.1 (i32.const 2408) "\02\00\00\00$\00\00\00 \00w\00i\00t\00h\00 \00p\00a\00r\00a\00m\00e\00t\00e\00r\00s\00:\00 ")
 (data $24 (i32.const 2460) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\18\00\00\000\t\00\00\00\00\00\00p\t\00\00\00\00\00\00\90\07")
 (data $25 (i32.const 2508) "\\")
 (data $25.1 (i32.const 2520) "\02\00\00\00D\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00 \00\"\00r\00e\00s\00u\00l\00t\00\"\00:\00 \00\"\00T\00o\00o\00l\00 ")
 (data $26 (i32.const 2604) "L")
 (data $26.1 (i32.const 2616) "\02\00\00\004\00\00\00 \00e\00x\00e\00c\00u\00t\00e\00d\00\"\00,\00 \00\"\00p\00a\00r\00a\00m\00e\00t\00e\00r\00s\00\"\00:\00 ")
 (data $27 (i32.const 2684) ",")
 (data $27.1 (i32.const 2696) "\02\00\00\00\1c\00\00\00,\00 \00\"\00c\00o\00n\00t\00e\00x\00t\00\"\00:\00 \00\"")
 (data $28 (i32.const 2732) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\1c\00\00\00\e0\t\00\00\00\00\00\00@\n\00\00\00\00\00\00\90\n\00\00\00\00\00\00\e0\08")
 (data $29 (i32.const 2780) "<")
 (data $29.1 (i32.const 2792) "\02\00\00\00,\00\00\00L\00o\00a\00d\00i\00n\00g\00 \00L\00o\00R\00A\00 \00a\00d\00a\00p\00t\00e\00r\00:\00 ")
 (data $30 (i32.const 2844) ",")
 (data $30.1 (i32.const 2856) "\02\00\00\00\1a\00\00\00{\00\"\00a\00g\00e\00n\00t\00I\00d\00\"\00:\00 \00\"")
 (data $31 (i32.const 2892) "<")
 (data $31.1 (i32.const 2904) "\02\00\00\00$\00\00\00\"\00,\00 \00\"\00i\00n\00i\00t\00i\00a\00l\00i\00z\00e\00d\00\"\00:\00 ")
 (data $32 (i32.const 2956) "<")
 (data $32.1 (i32.const 2968) "\02\00\00\00*\00\00\00,\00 \00\"\00v\00e\00r\00s\00i\00o\00n\00\"\00:\00 \00\"\001\00.\000\00.\000\00\"\00}")
 (data $33 (i32.const 3020) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\14\00\00\000\0b\00\00\00\00\00\00`\0b\00\00\00\00\00\00\a0\0b")
 (data $34 (i32.const 3068) "\1c")
 (data $34.1 (i32.const 3080) "\02\00\00\00\08\00\00\00t\00r\00u\00e")
 (data $35 (i32.const 3100) "\1c")
 (data $35.1 (i32.const 3112) "\02\00\00\00\n\00\00\00f\00a\00l\00s\00e")
 (data $36 (i32.const 3132) "L")
 (data $36.1 (i32.const 3144) "\02\00\00\000\00\00\00C\00r\00e\00a\00t\00i\00n\00g\00 \00n\00e\00w\00 \00M\00o\00d\00e\00l\00W\00A\00S\00M\00:\00 ")
 (data $37 (i32.const 3212) "L")
 (data $37.1 (i32.const 3224) "\02\00\00\006\00\00\00L\00o\00a\00d\00i\00n\00g\00 \00w\00e\00i\00g\00h\00t\00s\00 \00f\00o\00r\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $38 (i32.const 3292) ",")
 (data $38.1 (i32.const 3304) "\02\00\00\00\10\00\00\00 \00a\00t\00 \00p\00t\00r\00 ")
 (data $39 (i32.const 3340) "\1c")
 (data $39.1 (i32.const 3352) "\02\00\00\00\04\00\00\00 \00(")
 (data $40 (i32.const 3372) ",")
 (data $40.1 (i32.const 3384) "\02\00\00\00\0e\00\00\00 \00b\00y\00t\00e\00s\00)")
 (data $41 (i32.const 3420) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\1c\00\00\00\a0\0c\00\00\00\00\00\00\f0\0c\00\00\00\00\00\00 \r\00\00\00\00\00\00@\r")
 (data $42 (i32.const 3468) "\1c")
 (data $42.1 (i32.const 3480) "\02\00\00\00\06\00\00\000\00.\000")
 (data $43 (i32.const 3500) "\1c")
 (data $43.1 (i32.const 3512) "\02\00\00\00\06\00\00\00N\00a\00N")
 (data $44 (i32.const 3532) ",")
 (data $44.1 (i32.const 3544) "\02\00\00\00\12\00\00\00-\00I\00n\00f\00i\00n\00i\00t\00y")
 (data $45 (i32.const 3580) ",")
 (data $45.1 (i32.const 3592) "\02\00\00\00\10\00\00\00I\00n\00f\00i\00n\00i\00t\00y")
 (data $47 (i32.const 3688) "\88\02\1c\08\a0\d5\8f\fav\bf>\a2\7f\e1\ae\bav\acU0 \fb\16\8b\ea5\ce]J\89B\cf-;eU\aa\b0k\9a\dfE\1a=\03\cf\1a\e6\ca\c6\9a\c7\17\fep\abO\dc\bc\be\fc\b1w\ff\0c\d6kA\ef\91V\be<\fc\7f\90\ad\1f\d0\8d\83\9aU1(\\Q\d3\b5\c9\a6\ad\8f\acq\9d\cb\8b\ee#w\"\9c\eamSx@\91I\cc\aeW\ce\b6]y\12<\827V\fbM6\94\10\c2O\98H8o\ea\96\90\c7:\82%\cb\85t\d7\f4\97\bf\97\cd\cf\86\a0\e5\ac*\17\98\n4\ef\8e\b25*\fbg8\b2;?\c6\d2\df\d4\c8\84\ba\cd\d3\1a\'D\dd\c5\96\c9%\bb\ce\9fk\93\84\a5b}$l\ac\db\f6\da_\rXf\ab\a3&\f1\c3\de\93\f8\e2\f3\b8\80\ff\aa\a8\ad\b5\b5\8bJ|l\05_b\87S0\c14`\ff\bc\c9U&\ba\91\8c\85N\96\bd~)p$w\f9\df\8f\b8\e5\b8\9f\bd\df\a6\94}t\88\cf_\a9\f8\cf\9b\a8\8f\93pD\b9k\15\0f\bf\f8\f0\08\8a\b611eU%\b0\cd\ac\7f{\d0\c6\e2?\99\06;+*\c4\10\\\e4\d3\92si\99$$\aa\0e\ca\00\83\f2\b5\87\fd\eb\1a\11\92d\08\e5\bc\cc\88Po\t\cc\bc\8c,e\19\e2X\17\b7\d1\00\00\00\00\00\00@\9c\00\00\00\00\10\a5\d4\e8\00\00b\ac\c5\ebx\ad\84\t\94\f8x9?\81\b3\15\07\c9{\ce\97\c0p\\\ea{\ce2~\8fh\80\e9\ab\a48\d2\d5E\"\9a\17&\'O\9f\'\fb\c4\d41\a2c\ed\a8\ad\c8\8c8e\de\b0\dbe\ab\1a\8e\08\c7\83\9a\1dqB\f9\1d]\c4X\e7\1b\a6,iM\92\ea\8dp\1ad\ee\01\daJw\ef\9a\99\a3m\a2\85k}\b4{x\t\f2w\18\ddy\a1\e4T\b4\c2\c5\9b[\92\86[\86=]\96\c8\c5S5\c8\b3\a0\97\fa\\\b4*\95\e3_\a0\99\bd\9fF\de%\8c9\db4\c2\9b\a5\\\9f\98\a3r\9a\c6\f6\ce\be\e9TS\bf\dc\b7\e2A\"\f2\17\f3\fc\88\a5x\\\d3\9b\ce \cc\dfS!{\f3Z\16\98:0\1f\97\dc\b5\a0\e2\96\b3\e3\\S\d1\d9\a8<D\a7\a4\d9|\9b\fb\10D\a4\a7LLv\bb\1a\9c@\b6\ef\8e\ab\8b,\84W\a6\10\ef\1f\d0)1\91\e9\e5\a4\10\9b\9d\0c\9c\a1\fb\9b\10\e7)\f4;b\d9 (\ac\85\cf\a7z^KD\80-\dd\ac\03@\e4!\bf\8f\ffD^/\9cg\8eA\b8\8c\9c\9d\173\d4\a9\1b\e3\b4\92\db\19\9e\d9w\df\ban\bf\96\ebk\ee\f0\9b;\02\87\af")
 (data $48 (i32.const 4384) "<\fbW\fbr\fb\8c\fb\a7\fb\c1\fb\dc\fb\f6\fb\11\fc,\fcF\fca\fc{\fc\96\fc\b1\fc\cb\fc\e6\fc\00\fd\1b\fd5\fdP\fdk\fd\85\fd\a0\fd\ba\fd\d5\fd\ef\fd\n\fe%\fe?\feZ\fet\fe\8f\fe\a9\fe\c4\fe\df\fe\f9\fe\14\ff.\ffI\ffc\ff~\ff\99\ff\b3\ff\ce\ff\e8\ff\03\00\1e\008\00S\00m\00\88\00\a2\00\bd\00\d8\00\f2\00\r\01\'\01B\01\\\01w\01\92\01\ac\01\c7\01\e1\01\fc\01\16\021\02L\02f\02\81\02\9b\02\b6\02\d0\02\eb\02\06\03 \03;\03U\03p\03\8b\03\a5\03\c0\03\da\03\f5\03\0f\04*\04")
 (data $49 (i32.const 4560) "\01\00\00\00\n\00\00\00d\00\00\00\e8\03\00\00\10\'\00\00\a0\86\01\00@B\0f\00\80\96\98\00\00\e1\f5\05\00\ca\9a;")
 (data $50 (i32.const 4600) "0\000\000\001\000\002\000\003\000\004\000\005\000\006\000\007\000\008\000\009\001\000\001\001\001\002\001\003\001\004\001\005\001\006\001\007\001\008\001\009\002\000\002\001\002\002\002\003\002\004\002\005\002\006\002\007\002\008\002\009\003\000\003\001\003\002\003\003\003\004\003\005\003\006\003\007\003\008\003\009\004\000\004\001\004\002\004\003\004\004\004\005\004\006\004\007\004\008\004\009\005\000\005\001\005\002\005\003\005\004\005\005\005\006\005\007\005\008\005\009\006\000\006\001\006\002\006\003\006\004\006\005\006\006\006\007\006\008\006\009\007\000\007\001\007\002\007\003\007\004\007\005\007\006\007\007\007\008\007\009\008\000\008\001\008\002\008\003\008\004\008\005\008\006\008\007\008\008\008\009\009\000\009\001\009\002\009\003\009\004\009\005\009\006\009\007\009\008\009\009")
 (data $51 (i32.const 5004) "L")
 (data $51.1 (i32.const 5016) "\02\00\00\00:\00\00\00{\00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00M\00o\00d\00e\00l\00 \00n\00o\00t\00 \00l\00o\00a\00d\00e\00d\00\"\00}")
 (data $52 (i32.const 5084) "L")
 (data $52.1 (i32.const 5096) "\02\00\00\008\00\00\00R\00u\00n\00n\00i\00n\00g\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00o\00n\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $53 (i32.const 5164) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\10\00\00\00\f0\13\00\00\00\00\00\00\90\07")
 (data $54 (i32.const 5212) "\\")
 (data $54.1 (i32.const 5224) "\02\00\00\00F\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00 \00\"\00r\00e\00s\00u\00l\00t\00\"\00:\00 \00\"\00M\00o\00d\00e\00l\00 ")
 (data $55 (i32.const 5308) "L")
 (data $55.1 (i32.const 5320) "\02\00\00\00.\00\00\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00r\00e\00s\00u\00l\00t\00 \00f\00o\00r\00:\00 ")
 (data $56 (i32.const 5388) "<")
 (data $56.1 (i32.const 5400) "\02\00\00\00\"\00\00\00\"\00,\00 \00\"\00m\00o\00d\00e\00l\00T\00y\00p\00e\00\"\00:\00 \00\"")
 (data $57 (i32.const 5452) "<\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00$\00\00\00p\14\00\00\00\00\00\00\d0\14\00\00\00\00\00\00 \15\00\00\00\00\00\00\a0\08\00\00\00\00\00\00\e0\08")
 (data $58 (i32.const 5516) "<")
 (data $58.1 (i32.const 5528) "\02\00\00\00\1e\00\00\00{\00\"\00m\00o\00d\00e\00l\00T\00y\00p\00e\00\"\00:\00 \00\"")
 (data $59 (i32.const 5580) ",")
 (data $59.1 (i32.const 5592) "\02\00\00\00\1a\00\00\00\"\00,\00 \00\"\00l\00o\00a\00d\00e\00d\00\"\00:\00 ")
 (data $60 (i32.const 5628) "\ac")
 (data $60.1 (i32.const 5640) "\02\00\00\00\92\00\00\00,\00 \00\"\00c\00a\00p\00a\00b\00i\00l\00i\00t\00i\00e\00s\00\"\00:\00 \00[\00\"\00t\00e\00x\00t\00-\00g\00e\00n\00e\00r\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00,\00 \00\"\00e\00x\00t\00e\00r\00n\00a\00l\00-\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00]\00}")
 (data $61 (i32.const 5804) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\14\00\00\00\a0\15\00\00\00\00\00\00\e0\15\00\00\00\00\00\00\10\16")
 (data $62 (i32.const 5852) "l")
 (data $62.1 (i32.const 5864) "\02\00\00\00Z\00\00\00C\00o\00n\00f\00i\00g\00u\00r\00i\00n\00g\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00f\00o\00r\00 \00p\00r\00o\00v\00i\00d\00e\00r\00:\00 ")
 (data $63 (i32.const 5964) ",")
 (data $63.1 (i32.const 5976) "\02\00\00\00\12\00\00\00P\00r\00o\00v\00i\00d\00e\00r\00 ")
 (data $64 (i32.const 6012) "L")
 (data $64.1 (i32.const 6024) "\02\00\00\006\00\00\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00 \00w\00i\00t\00h\00 \00e\00n\00d\00p\00o\00i\00n\00t\00:\00 ")
 (data $65 (i32.const 6092) ",")
 (data $65.1 (i32.const 6104) "\02\00\00\00\12\00\00\00,\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $66 (i32.const 6140) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\18\00\00\00`\17\00\00\00\00\00\00\90\17\00\00\00\00\00\00\e0\17")
 (data $67 (i32.const 6188) "\\")
 (data $67.1 (i32.const 6200) "\02\00\00\00D\00\00\00A\00c\00t\00i\00v\00e\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00p\00r\00o\00v\00i\00d\00e\00r\00 \00s\00e\00t\00 \00t\00o\00:\00 ")
 (data $68 (i32.const 6284) "<")
 (data $68.1 (i32.const 6296) "\02\00\00\00\1e\00\00\00 \00n\00o\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d")
 (data $69 (i32.const 6348) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00`\17\00\00\00\00\00\00\a0\18")
 (data $70 (i32.const 6380) "\1c")
 (data $70.1 (i32.const 6392) "\01")
 (data $71 (i32.const 6412) "\1c")
 (data $71.1 (i32.const 6424) "\02\00\00\00\0c\00\00\00g\00e\00m\00i\00n\00i")
 (data $72 (i32.const 6444) ",")
 (data $72.1 (i32.const 6456) "\02\00\00\00\1a\00\00\00~\00l\00i\00b\00/\00a\00r\00r\00a\00y\00.\00t\00s")
 (data $73 (i32.const 6492) ",")
 (data $73.1 (i32.const 6504) "\02\00\00\00\10\00\00\00c\00e\00r\00e\00b\00r\00a\00s")
 (data $74 (i32.const 6540) ",")
 (data $74.1 (i32.const 6552) "\02\00\00\00\10\00\00\00d\00e\00e\00p\00s\00e\00e\00k")
 (data $75 (i32.const 6588) "\1c")
 (data $75.1 (i32.const 6600) "\02\00\00\00\0c\00\00\00c\00l\00a\00u\00d\00e")
 (data $76 (i32.const 6620) "\1c")
 (data $76.1 (i32.const 6632) "\02\00\00\00\0c\00\00\00o\00p\00e\00n\00a\00i")
 (data $77 (i32.const 6652) "<")
 (data $77.1 (i32.const 6664) "\02\00\00\00\1e\00\00\00{\00\"\00p\00r\00o\00v\00i\00d\00e\00r\00s\00\"\00:\00 \00[")
 (data $78 (i32.const 6716) "\1c")
 (data $78.1 (i32.const 6728) "\02\00\00\00\04\00\00\00]\00}")
 (data $79 (i32.const 6748) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\10\1a\00\00\00\00\00\00P\1a")
 (data $80 (i32.const 6780) "\1c")
 (data $80.1 (i32.const 6792) "\02\00\00\00\02\00\00\00\"")
 (data $81 (i32.const 6812) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\90\1a\00\00\00\00\00\00\90\1a")
 (data $82 (i32.const 6844) "\1c")
 (data $82.1 (i32.const 6856) "\08\00\00\00\08\00\00\00\01")
 (data $83 (i32.const 6876) "\1c")
 (data $83.1 (i32.const 6888) "\02\00\00\00\04\00\00\00,\00 ")
 (data $84 (i32.const 6908) "\9c")
 (data $84.1 (i32.const 6920) "\02\00\00\00\80\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00f\00a\00l\00s\00e\00,\00 \00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00E\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00n\00o\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00\"\00}")
 (data $85 (i32.const 7068) "\8c")
 (data $85.1 (i32.const 7080) "\02\00\00\00z\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00f\00a\00l\00s\00e\00,\00 \00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00A\00c\00t\00i\00v\00e\00 \00p\00r\00o\00v\00i\00d\00e\00r\00 \00n\00o\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00\"\00}")
 (data $86 (i32.const 7212) "<")
 (data $86.1 (i32.const 7224) "\02\00\00\00$\00\00\00K\00e\00y\00 \00d\00o\00e\00s\00 \00n\00o\00t\00 \00e\00x\00i\00s\00t")
 (data $87 (i32.const 7276) ",")
 (data $87.1 (i32.const 7288) "\02\00\00\00\16\00\00\00~\00l\00i\00b\00/\00m\00a\00p\00.\00t\00s")
 (data $88 (i32.const 7324) "\\")
 (data $88.1 (i32.const 7336) "\02\00\00\00F\00\00\00P\00e\00r\00f\00o\00r\00m\00i\00n\00g\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00w\00i\00t\00h\00 ")
 (data $89 (i32.const 7420) ",")
 (data $89.1 (i32.const 7432) "\02\00\00\00\1c\00\00\00 \00u\00s\00i\00n\00g\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $90 (i32.const 7468) ",")
 (data $90.1 (i32.const 7480) "\02\00\00\00\1a\00\00\00,\00 \00m\00a\00x\00T\00o\00k\00e\00n\00s\00:\00 ")
 (data $91 (i32.const 7516) "<")
 (data $91.1 (i32.const 7528) "\02\00\00\00\1e\00\00\00,\00 \00t\00e\00m\00p\00e\00r\00a\00t\00u\00r\00e\00:\00 ")
 (data $92 (i32.const 7580) "<\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00 \00\00\00\b0\1c\00\00\00\00\00\00\10\1d\00\00\00\00\00\00@\1d\00\00\00\00\00\00p\1d")
 (data $93 (i32.const 7644) "L")
 (data $93.1 (i32.const 7656) "\02\00\00\000\00\00\00G\00e\00m\00i\00n\00i\00 \00A\00I\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $94 (i32.const 7724) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\f0\1d\00\00\00\00\00\00\90\1a")
 (data $95 (i32.const 7756) "L")
 (data $95.1 (i32.const 7768) "\02\00\00\008\00\00\00 \00(\00f\00o\00l\00l\00o\00w\00i\00n\00g\00 \00s\00y\00s\00t\00e\00m\00 \00p\00r\00o\00m\00p\00t\00:\00 \00\"")
 (data $96 (i32.const 7836) "\1c")
 (data $96.1 (i32.const 7848) "\02\00\00\00\04\00\00\00\"\00)")
 (data $97 (i32.const 7868) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00`\1e\00\00\00\00\00\00\b0\1e")
 (data $98 (i32.const 7900) "|")
 (data $98.1 (i32.const 7912) "\02\00\00\00^\00\00\00C\00e\00r\00e\00b\00r\00a\00s\00 \00h\00i\00g\00h\00-\00p\00e\00r\00f\00o\00r\00m\00a\00n\00c\00e\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00r\00e\00s\00p\00o\00n\00s\00e\00:\00 \00\"")
 (data $99 (i32.const 8028) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\f0\1e\00\00\00\00\00\00\90\1a")
 (data $100 (i32.const 8060) ",")
 (data $100.1 (i32.const 8072) "\02\00\00\00\16\00\00\00 \00(\00s\00y\00s\00t\00e\00m\00:\00 \00\"")
 (data $101 (i32.const 8108) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\90\1f\00\00\00\00\00\00\b0\1e")
 (data $102 (i32.const 8140) "\\")
 (data $102.1 (i32.const 8152) "\02\00\00\00B\00\00\00D\00e\00e\00p\00S\00e\00e\00k\00 \00r\00e\00a\00s\00o\00n\00i\00n\00g\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $103 (i32.const 8236) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\e0\1f\00\00\00\00\00\00\90\1a")
 (data $104 (i32.const 8268) ",")
 (data $104.1 (i32.const 8280) "\02\00\00\00\1c\00\00\00 \00(\00g\00u\00i\00d\00e\00d\00 \00b\00y\00:\00 \00\"")
 (data $105 (i32.const 8316) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00` \00\00\00\00\00\00\b0\1e")
 (data $106 (i32.const 8348) "\\")
 (data $106.1 (i32.const 8360) "\02\00\00\00H\00\00\00C\00l\00a\00u\00d\00e\00 \00c\00o\00n\00s\00t\00i\00t\00u\00t\00i\00o\00n\00a\00l\00 \00A\00I\00 \00r\00e\00s\00p\00o\00n\00s\00e\00:\00 \00\"")
 (data $107 (i32.const 8444) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\b0 \00\00\00\00\00\00\90\1a")
 (data $108 (i32.const 8476) "L")
 (data $108.1 (i32.const 8488) "\02\00\00\002\00\00\00 \00(\00w\00i\00t\00h\00 \00s\00y\00s\00t\00e\00m\00 \00g\00u\00i\00d\00a\00n\00c\00e\00:\00 \00\"")
 (data $109 (i32.const 8556) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\000!\00\00\00\00\00\00\b0\1e")
 (data $110 (i32.const 8588) "L")
 (data $110.1 (i32.const 8600) "\02\00\00\002\00\00\00O\00p\00e\00n\00A\00I\00 \00G\00P\00T\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $111 (i32.const 8668) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\a0!\00\00\00\00\00\00\90\1a")
 (data $112 (i32.const 8700) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\90\1f\00\00\00\00\00\00\b0\1e")
 (data $113 (i32.const 8732) "\\")
 (data $113.1 (i32.const 8744) "\02\00\00\00>\00\00\00U\00n\00k\00n\00o\00w\00n\00 \00p\00r\00o\00v\00i\00d\00e\00r\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $114 (i32.const 8828) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\000\"\00\00\00\00\00\00\90\1a")
 (data $115 (i32.const 8860) "l")
 (data $115.1 (i32.const 8872) "\02\00\00\00N\00\00\00{\00\n\00 \00 \00 \00 \00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00\n\00 \00 \00 \00 \00\"\00c\00o\00n\00t\00e\00n\00t\00\"\00:\00 \00\"")
 (data $116 (i32.const 8972) "<")
 (data $116.1 (i32.const 8984) "\02\00\00\00(\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00p\00r\00o\00v\00i\00d\00e\00r\00\"\00:\00 \00\"")
 (data $117 (i32.const 9036) "<")
 (data $117.1 (i32.const 9048) "\02\00\00\00\"\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00m\00o\00d\00e\00l\00\"\00:\00 \00\"")
 (data $118 (i32.const 9100) "<")
 (data $118.1 (i32.const 9112) "\02\00\00\00(\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00m\00a\00x\00T\00o\00k\00e\00n\00s\00\"\00:\00 ")
 (data $119 (i32.const 9164) "<")
 (data $119.1 (i32.const 9176) "\02\00\00\00*\00\00\00,\00\n\00 \00 \00 \00 \00\"\00t\00e\00m\00p\00e\00r\00a\00t\00u\00r\00e\00\"\00:\00 ")
 (data $120 (i32.const 9228) "\9c")
 (data $120.1 (i32.const 9240) "\02\00\00\00\88\00\00\00,\00\n\00 \00 \00 \00 \00\"\00p\00r\00o\00c\00e\00s\00s\00i\00n\00g\00T\00i\00m\00e\00\"\00:\00 \001\005\000\00.\005\00,\00\n\00 \00 \00 \00 \00\"\00u\00s\00a\00g\00e\00\"\00:\00 \00{\00\n\00 \00 \00 \00 \00 \00 \00\"\00p\00r\00o\00m\00p\00t\00T\00o\00k\00e\00n\00s\00\"\00:\00 ")
 (data $121 (i32.const 9388) "L")
 (data $121.1 (i32.const 9400) "\02\00\00\008\00\00\00,\00\n\00 \00 \00 \00 \00 \00 \00\"\00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00T\00o\00k\00e\00n\00s\00\"\00:\00 ")
 (data $122 (i32.const 9468) "L")
 (data $122.1 (i32.const 9480) "\02\00\00\00.\00\00\00,\00\n\00 \00 \00 \00 \00 \00 \00\"\00t\00o\00t\00a\00l\00T\00o\00k\00e\00n\00s\00\"\00:\00 ")
 (data $123 (i32.const 9548) ",")
 (data $123.1 (i32.const 9560) "\02\00\00\00\14\00\00\00\n\00 \00 \00 \00 \00}\00\n\00 \00 \00}")
 (data $124 (i32.const 9596) "\\\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00D\00\00\00\b0\"\00\00\00\00\00\00 #\00\00\00\00\00\00`#\00\00\00\00\00\00\a0#\00\00\00\00\00\00\e0#\00\00\00\00\00\00 $\00\00\00\00\00\00\c0$\00\00\00\00\00\00\10%\00\00\00\00\00\00`%")
 (data $125 (i32.const 9692) "|")
 (data $125.1 (i32.const 9704) "\02\00\00\00d\00\00\00t\00o\00S\00t\00r\00i\00n\00g\00(\00)\00 \00r\00a\00d\00i\00x\00 \00a\00r\00g\00u\00m\00e\00n\00t\00 \00m\00u\00s\00t\00 \00b\00e\00 \00b\00e\00t\00w\00e\00e\00n\00 \002\00 \00a\00n\00d\00 \003\006")
 (data $126 (i32.const 9820) "<")
 (data $126.1 (i32.const 9832) "\02\00\00\00&\00\00\00~\00l\00i\00b\00/\00u\00t\00i\00l\00/\00n\00u\00m\00b\00e\00r\00.\00t\00s")
 (data $127 (i32.const 9884) "\1c")
 (data $127.1 (i32.const 9896) "\02\00\00\00\02\00\00\000")
 (data $128 (i32.const 9916) "\1c\04")
 (data $128.1 (i32.const 9928) "\02\00\00\00\00\04\00\000\000\000\001\000\002\000\003\000\004\000\005\000\006\000\007\000\008\000\009\000\00a\000\00b\000\00c\000\00d\000\00e\000\00f\001\000\001\001\001\002\001\003\001\004\001\005\001\006\001\007\001\008\001\009\001\00a\001\00b\001\00c\001\00d\001\00e\001\00f\002\000\002\001\002\002\002\003\002\004\002\005\002\006\002\007\002\008\002\009\002\00a\002\00b\002\00c\002\00d\002\00e\002\00f\003\000\003\001\003\002\003\003\003\004\003\005\003\006\003\007\003\008\003\009\003\00a\003\00b\003\00c\003\00d\003\00e\003\00f\004\000\004\001\004\002\004\003\004\004\004\005\004\006\004\007\004\008\004\009\004\00a\004\00b\004\00c\004\00d\004\00e\004\00f\005\000\005\001\005\002\005\003\005\004\005\005\005\006\005\007\005\008\005\009\005\00a\005\00b\005\00c\005\00d\005\00e\005\00f\006\000\006\001\006\002\006\003\006\004\006\005\006\006\006\007\006\008\006\009\006\00a\006\00b\006\00c\006\00d\006\00e\006\00f\007\000\007\001\007\002\007\003\007\004\007\005\007\006\007\007\007\008\007\009\007\00a\007\00b\007\00c\007\00d\007\00e\007\00f\008\000\008\001\008\002\008\003\008\004\008\005\008\006\008\007\008\008\008\009\008\00a\008\00b\008\00c\008\00d\008\00e\008\00f\009\000\009\001\009\002\009\003\009\004\009\005\009\006\009\007\009\008\009\009\009\00a\009\00b\009\00c\009\00d\009\00e\009\00f\00a\000\00a\001\00a\002\00a\003\00a\004\00a\005\00a\006\00a\007\00a\008\00a\009\00a\00a\00a\00b\00a\00c\00a\00d\00a\00e\00a\00f\00b\000\00b\001\00b\002\00b\003\00b\004\00b\005\00b\006\00b\007\00b\008\00b\009\00b\00a\00b\00b\00b\00c\00b\00d\00b\00e\00b\00f\00c\000\00c\001\00c\002\00c\003\00c\004\00c\005\00c\006\00c\007\00c\008\00c\009\00c\00a\00c\00b\00c\00c\00c\00d\00c\00e\00c\00f\00d\000\00d\001\00d\002\00d\003\00d\004\00d\005\00d\006\00d\007\00d\008\00d\009\00d\00a\00d\00b\00d\00c\00d\00d\00d\00e\00d\00f\00e\000\00e\001\00e\002\00e\003\00e\004\00e\005\00e\006\00e\007\00e\008\00e\009\00e\00a\00e\00b\00e\00c\00e\00d\00e\00e\00e\00f\00f\000\00f\001\00f\002\00f\003\00f\004\00f\005\00f\006\00f\007\00f\008\00f\009\00f\00a\00f\00b\00f\00c\00f\00d\00f\00e\00f\00f")
 (data $129 (i32.const 10972) "\\")
 (data $129.1 (i32.const 10984) "\02\00\00\00H\00\00\000\001\002\003\004\005\006\007\008\009\00a\00b\00c\00d\00e\00f\00g\00h\00i\00j\00k\00l\00m\00n\00o\00p\00q\00r\00s\00t\00u\00v\00w\00x\00y\00z")
 (data $130 (i32.const 11068) "<")
 (data $130.1 (i32.const 11080) "\02\00\00\00\"\00\00\00{\00\n\00 \00 \00 \00 \00\"\00e\00n\00a\00b\00l\00e\00d\00\"\00:\00 ")
 (data $131 (i32.const 11132) "L")
 (data $131.1 (i32.const 11144) "\02\00\00\002\00\00\00,\00\n\00 \00 \00 \00 \00\"\00a\00c\00t\00i\00v\00e\00P\00r\00o\00v\00i\00d\00e\00r\00\"\00:\00 \00\"")
 (data $132 (i32.const 11212) "L")
 (data $132.1 (i32.const 11224) "\02\00\00\00<\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00P\00r\00o\00v\00i\00d\00e\00r\00s\00\"\00:\00 ")
 (data $133 (i32.const 11292) "\bc")
 (data $133.1 (i32.const 11304) "\02\00\00\00\a0\00\00\00,\00\n\00 \00 \00 \00 \00\"\00c\00a\00p\00a\00b\00i\00l\00i\00t\00i\00e\00s\00\"\00:\00 \00[\00\"\00c\00h\00a\00t\00-\00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00\"\00,\00 \00\"\00t\00e\00x\00t\00-\00g\00e\00n\00e\00r\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00c\00o\00n\00v\00e\00r\00s\00a\00t\00i\00o\00n\00\"\00]\00\n\00 \00 \00}")
 (data $134 (i32.const 11484) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\1c\00\00\00P+\00\00\00\00\00\00\90+\00\00\00\00\00\00\e0+\00\00\00\00\00\000,")
 (data $135 (i32.const 11532) "\1c")
 (data $135.1 (i32.const 11544) "\02\00\00\00\n\00\00\001\00.\000\00.\000")
 (data $136 (i32.const 11564) "<\01")
 (data $136.1 (i32.const 11576) "\02\00\00\00&\01\00\00[\00\"\00a\00g\00e\00n\00t\00-\00c\00o\00r\00e\00\"\00,\00 \00\"\00m\00o\00d\00e\00l\00-\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00,\00 \00\"\00l\00o\00r\00a\00-\00a\00d\00a\00p\00t\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00c\00r\00o\00s\00s\00-\00w\00a\00s\00m\00-\00c\00o\00m\00m\00u\00n\00i\00c\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00e\00x\00t\00e\00r\00n\00a\00l\00-\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00,\00 \00\"\00c\00h\00a\00t\00-\00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00\"\00,\00 \00\"\00m\00u\00l\00t\00i\00-\00p\00r\00o\00v\00i\00d\00e\00r\00-\00s\00u\00p\00p\00o\00r\00t\00\"\00]")
 (data $137 (i32.const 11884) "\1c")
 (data $137.1 (i32.const 11896) "\02\00\00\00\04\00\00\00{\00}")
 (data $138 (i32.const 11916) "\8c")
 (data $138.1 (i32.const 11928) "\02\00\00\00x\00\00\00P\00e\00r\00f\00o\00r\00m\00i\00n\00g\00 \00c\00h\00a\00t\00 \00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00 \00w\00i\00t\00h\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00,\00 \00c\00o\00n\00f\00i\00g\00:\00 ")
 (data $139 (i32.const 12060) ",")
 (data $139.1 (i32.const 12072) "\02\00\00\00\1a\00\00\00\"\00r\00o\00l\00e\00\"\00:\00\"\00u\00s\00e\00r\00\"")
 (data $140 (i32.const 12108) ",")
 (data $140.1 (i32.const 12120) "\02\00\00\00\16\00\00\00\"\00c\00o\00n\00t\00e\00n\00t\00\"\00:\00\"")
 (data $141 (i32.const 12156) "<")
 (data $141.1 (i32.const 12168) "\02\00\00\00\1e\00\00\00\"\00r\00o\00l\00e\00\"\00:\00\"\00s\00y\00s\00t\00e\00m\00\"")
 (data $142 (i32.const 12220) "\9c")
 (data $142.1 (i32.const 12232) "\02\00\00\00\88\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00f\00a\00l\00s\00e\00,\00 \00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00N\00o\00 \00u\00s\00e\00r\00 \00m\00e\00s\00s\00a\00g\00e\00 \00f\00o\00u\00n\00d\00 \00i\00n\00 \00c\00o\00n\00v\00e\00r\00s\00a\00t\00i\00o\00n\00\"\00}")
 (data $143 (i32.const 12380) "\9c")
 (data $143.1 (i32.const 12392) "\02\00\00\00\80\00\00\00I\00n\00i\00t\00i\00a\00l\00i\00z\00i\00n\00g\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00f\00r\00o\00m\00 \00e\00n\00v\00i\00r\00o\00n\00m\00e\00n\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00a\00t\00i\00o\00n\00:\00 ")
 (data $144 (i32.const 12540) "<")
 (data $144.1 (i32.const 12552) "\02\00\00\00\1e\00\00\00t\00e\00s\00t\00-\00g\00e\00m\00i\00n\00i\00-\00k\00e\00y")
 (data $145 (i32.const 12604) "|")
 (data $145.1 (i32.const 12616) "\02\00\00\00`\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00g\00e\00n\00e\00r\00a\00t\00i\00v\00e\00l\00a\00n\00g\00u\00a\00g\00e\00.\00g\00o\00o\00g\00l\00e\00a\00p\00i\00s\00.\00c\00o\00m\00/\00v\001\00b\00e\00t\00a")
 (data $146 (i32.const 12732) ",")
 (data $146.1 (i32.const 12744) "\02\00\00\00\14\00\00\00g\00e\00m\00i\00n\00i\00-\00p\00r\00o")
 (data $147 (i32.const 12780) "<")
 (data $147.1 (i32.const 12792) "\02\00\00\00\"\00\00\00t\00e\00s\00t\00-\00c\00e\00r\00e\00b\00r\00a\00s\00-\00k\00e\00y")
 (data $148 (i32.const 12844) "L")
 (data $148.1 (i32.const 12856) "\02\00\00\004\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00c\00e\00r\00e\00b\00r\00a\00s\00.\00a\00i\00/\00v\001")
 (data $149 (i32.const 12924) ",")
 (data $149.1 (i32.const 12936) "\02\00\00\00\16\00\00\00l\00l\00a\00m\00a\003\00.\001\00-\008\00b")
 (data $150 (i32.const 12972) "<")
 (data $150.1 (i32.const 12984) "\02\00\00\00\"\00\00\00t\00e\00s\00t\00-\00d\00e\00e\00p\00s\00e\00e\00k\00-\00k\00e\00y")
 (data $151 (i32.const 13036) "L")
 (data $151.1 (i32.const 13048) "\02\00\00\006\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00d\00e\00e\00p\00s\00e\00e\00k\00.\00c\00o\00m\00/\00v\001")
 (data $152 (i32.const 13116) ",")
 (data $152.1 (i32.const 13128) "\02\00\00\00\1a\00\00\00d\00e\00e\00p\00s\00e\00e\00k\00-\00c\00h\00a\00t")
 (data $153 (i32.const 13164) "<")
 (data $153.1 (i32.const 13176) "\02\00\00\00\1e\00\00\00t\00e\00s\00t\00-\00c\00l\00a\00u\00d\00e\00-\00k\00e\00y")
 (data $154 (i32.const 13228) "L")
 (data $154.1 (i32.const 13240) "\02\00\00\008\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00a\00n\00t\00h\00r\00o\00p\00i\00c\00.\00c\00o\00m\00/\00v\001")
 (data $155 (i32.const 13308) "<")
 (data $155.1 (i32.const 13320) "\02\00\00\00\1e\00\00\00c\00l\00a\00u\00d\00e\00-\003\00-\00s\00o\00n\00n\00e\00t")
 (data $156 (i32.const 13372) "<")
 (data $156.1 (i32.const 13384) "\02\00\00\00\1e\00\00\00t\00e\00s\00t\00-\00o\00p\00e\00n\00a\00i\00-\00k\00e\00y")
 (data $157 (i32.const 13436) "L")
 (data $157.1 (i32.const 13448) "\02\00\00\002\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00o\00p\00e\00n\00a\00i\00.\00c\00o\00m\00/\00v\001")
 (data $158 (i32.const 13516) "\1c")
 (data $158.1 (i32.const 13528) "\02\00\00\00\n\00\00\00g\00p\00t\00-\004")
 (data $159 (i32.const 13548) "|")
 (data $159.1 (i32.const 13560) "\02\00\00\00l\00\00\00E\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00i\00n\00i\00t\00i\00a\00l\00i\00z\00e\00d\00 \00w\00i\00t\00h\00 \00m\00u\00l\00t\00i\00p\00l\00e\00 \00p\00r\00o\00v\00i\00d\00e\00r\00s")
 (data $160 (i32.const 13676) "\8c")
 (data $160.1 (i32.const 13688) "\02\00\00\00n\00\00\00K\00N\00I\00R\00V\00 \00C\00o\00n\00t\00r\00o\00l\00l\00e\00r\00 \00A\00s\00s\00e\00m\00b\00l\00y\00S\00c\00r\00i\00p\00t\00 \00W\00A\00S\00M\00 \00m\00o\00d\00u\00l\00e\00 \00i\00n\00i\00t\00i\00a\00l\00i\00z\00e\00d")
 (data $161 (i32.const 13820) "<")
 (data $161.1 (i32.const 13832) "\02\00\00\00*\00\00\00O\00b\00j\00e\00c\00t\00 \00a\00l\00r\00e\00a\00d\00y\00 \00p\00i\00n\00n\00e\00d")
 (data $162 (i32.const 13884) "<")
 (data $162.1 (i32.const 13896) "\02\00\00\00(\00\00\00O\00b\00j\00e\00c\00t\00 \00i\00s\00 \00n\00o\00t\00 \00p\00i\00n\00n\00e\00d")
 (data $163 (i32.const 13952) "\t\00\00\00 \00\00\00 \00\00\00 \00\00\00\00\00\00\00\10A\82\00\04A\00\00\02A\00\00\02\t")
 (table $0 2 2 funcref)
 (elem $0 (i32.const 1) $assembly/index/getConfiguredProviders~anonymous|0)
 (export "initializeAgent" (func $assembly/index/initializeAgent))
 (export "getAgentStatus" (func $assembly/index/getAgentStatus))
 (export "loadModelWeights" (func $assembly/index/loadModelWeights))
 (export "getModelInfo" (func $assembly/index/getModelInfo))
 (export "getConfiguredProviders" (func $assembly/index/getConfiguredProviders))
 (export "getExternalInferenceStatus" (func $assembly/index/getExternalInferenceStatus))
 (export "getWasmVersion" (func $assembly/index/getWasmVersion))
 (export "getSupportedFeatures" (func $assembly/index/getSupportedFeatures))
 (export "deallocateString" (func $assembly/index/deallocateString))
 (export "wasmInit" (func $assembly/index/wasmInit))
 (export "__new" (func $~lib/rt/itcms/__new))
 (export "__pin" (func $~lib/rt/itcms/__pin))
 (export "__unpin" (func $~lib/rt/itcms/__unpin))
 (export "__collect" (func $~lib/rt/itcms/__collect))
 (export "__rtti_base" (global $~lib/rt/__rtti_base))
 (export "memory" (memory $0))
 (export "__setArgumentsLength" (func $~setArgumentsLength))
 (export "createAgentCore" (func $export:assembly/index/createAgentCore))
 (export "executeAgent" (func $export:assembly/index/executeAgent))
 (export "executeAgentTool" (func $export:assembly/index/executeAgentTool))
 (export "loadLoraAdapter" (func $export:assembly/index/loadLoraAdapter))
 (export "createModel" (func $export:assembly/index/createModel))
 (export "runModelInference" (func $export:assembly/index/runModelInference))
 (export "configureExternalInference" (func $export:assembly/index/configureExternalInference))
 (export "setActiveInferenceProvider" (func $export:assembly/index/setActiveInferenceProvider))
 (export "performExternalInference" (func $export:assembly/index/performExternalInference@varargs))
 (export "performChatCompletion" (func $export:assembly/index/performChatCompletion@varargs))
 (export "initializeExternalInferenceFromEnv" (func $export:assembly/index/initializeExternalInferenceFromEnv))
 (export "allocateString" (func $export:assembly/index/allocateString))
 (start $~start)
 (func $~lib/rt/itcms/visitRoots
  (local $0 i32)
  (local $1 i32)
  global.get $assembly/index/agentId
  local.tee $0
  if
   local.get $0
   call $~lib/rt/itcms/__visit
  end
  global.get $assembly/index/modelType
  local.tee $0
  if
   local.get $0
   call $~lib/rt/itcms/__visit
  end
  global.get $assembly/index/activeProvider
  local.tee $0
  if
   local.get $0
   call $~lib/rt/itcms/__visit
  end
  global.get $assembly/index/apiKeys
  local.tee $0
  if
   local.get $0
   call $~lib/rt/itcms/__visit
  end
  global.get $assembly/index/providerEndpoints
  local.tee $0
  if
   local.get $0
   call $~lib/rt/itcms/__visit
  end
  global.get $assembly/index/providerModels
  local.tee $0
  if
   local.get $0
   call $~lib/rt/itcms/__visit
  end
  i32.const 1280
  call $~lib/rt/itcms/__visit
  i32.const 1488
  call $~lib/rt/itcms/__visit
  i32.const 7232
  call $~lib/rt/itcms/__visit
  i32.const 1088
  call $~lib/rt/itcms/__visit
  i32.const 13840
  call $~lib/rt/itcms/__visit
  i32.const 13904
  call $~lib/rt/itcms/__visit
  i32.const 9936
  call $~lib/rt/itcms/__visit
  i32.const 10992
  call $~lib/rt/itcms/__visit
  global.get $~lib/rt/itcms/pinSpace
  local.tee $1
  i32.load offset=4
  i32.const -4
  i32.and
  local.set $0
  loop $while-continue|0
   local.get $0
   local.get $1
   i32.ne
   if
    local.get $0
    i32.load offset=4
    drop
    local.get $0
    i32.const 20
    i32.add
    call $~lib/rt/__visit_members
    local.get $0
    i32.load offset=4
    i32.const -4
    i32.and
    local.set $0
    br $while-continue|0
   end
  end
 )
 (func $~lib/rt/itcms/Object#makeGray (param $0 i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  local.get $0
  global.get $~lib/rt/itcms/iter
  i32.eq
  if
   local.get $0
   i32.load offset=8
   global.set $~lib/rt/itcms/iter
  end
  block $__inlined_func$~lib/rt/itcms/Object#unlink
   local.get $0
   i32.load offset=4
   i32.const -4
   i32.and
   local.tee $1
   i32.eqz
   if
    local.get $0
    i32.load offset=8
    drop
    br $__inlined_func$~lib/rt/itcms/Object#unlink
   end
   local.get $1
   local.get $0
   i32.load offset=8
   local.tee $2
   i32.store offset=8
   local.get $2
   local.get $1
   local.get $2
   i32.load offset=4
   i32.const 3
   i32.and
   i32.or
   i32.store offset=4
  end
  global.get $~lib/rt/itcms/toSpace
  local.set $2
  local.get $0
  i32.load offset=12
  local.tee $1
  i32.const 2
  i32.le_u
  if (result i32)
   i32.const 1
  else
   local.get $1
   i32.const 13952
   i32.load
   i32.gt_u
   if
    i32.const 1280
    i32.const 1344
    i32.const 21
    i32.const 28
    call $~lib/builtins/abort
    unreachable
   end
   local.get $1
   i32.const 2
   i32.shl
   i32.const 13956
   i32.add
   i32.load
   i32.const 32
   i32.and
  end
  local.set $3
  local.get $2
  i32.load offset=8
  local.set $1
  local.get $0
  global.get $~lib/rt/itcms/white
  i32.eqz
  i32.const 2
  local.get $3
  select
  local.get $2
  i32.or
  i32.store offset=4
  local.get $0
  local.get $1
  i32.store offset=8
  local.get $1
  local.get $0
  local.get $1
  i32.load offset=4
  i32.const 3
  i32.and
  i32.or
  i32.store offset=4
  local.get $2
  local.get $0
  i32.store offset=8
 )
 (func $~lib/rt/itcms/__visit (param $0 i32)
  local.get $0
  i32.eqz
  if
   return
  end
  global.get $~lib/rt/itcms/white
  local.get $0
  i32.const 20
  i32.sub
  local.tee $0
  i32.load offset=4
  i32.const 3
  i32.and
  i32.eq
  if
   local.get $0
   call $~lib/rt/itcms/Object#makeGray
   global.get $~lib/rt/itcms/visitCount
   i32.const 1
   i32.add
   global.set $~lib/rt/itcms/visitCount
  end
 )
 (func $~lib/rt/tlsf/removeBlock (param $0 i32) (param $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  local.get $1
  i32.load
  i32.const -4
  i32.and
  local.tee $3
  i32.const 256
  i32.lt_u
  if (result i32)
   local.get $3
   i32.const 4
   i32.shr_u
  else
   i32.const 31
   i32.const 1073741820
   local.get $3
   local.get $3
   i32.const 1073741820
   i32.ge_u
   select
   local.tee $3
   i32.clz
   i32.sub
   local.tee $4
   i32.const 7
   i32.sub
   local.set $2
   local.get $3
   local.get $4
   i32.const 4
   i32.sub
   i32.shr_u
   i32.const 16
   i32.xor
  end
  local.set $4
  local.get $1
  i32.load offset=8
  local.set $5
  local.get $1
  i32.load offset=4
  local.tee $3
  if
   local.get $3
   local.get $5
   i32.store offset=8
  end
  local.get $5
  if
   local.get $5
   local.get $3
   i32.store offset=4
  end
  local.get $1
  local.get $0
  local.get $2
  i32.const 4
  i32.shl
  local.get $4
  i32.add
  i32.const 2
  i32.shl
  i32.add
  local.tee $1
  i32.load offset=96
  i32.eq
  if
   local.get $1
   local.get $5
   i32.store offset=96
   local.get $5
   i32.eqz
   if
    local.get $0
    local.get $2
    i32.const 2
    i32.shl
    i32.add
    local.tee $1
    i32.load offset=4
    i32.const -2
    local.get $4
    i32.rotl
    i32.and
    local.set $3
    local.get $1
    local.get $3
    i32.store offset=4
    local.get $3
    i32.eqz
    if
     local.get $0
     local.get $0
     i32.load
     i32.const -2
     local.get $2
     i32.rotl
     i32.and
     i32.store
    end
   end
  end
 )
 (func $~lib/rt/tlsf/insertBlock (param $0 i32) (param $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  local.get $1
  i32.const 4
  i32.add
  local.tee $6
  local.get $1
  i32.load
  local.tee $3
  i32.const -4
  i32.and
  i32.add
  local.tee $4
  i32.load
  local.tee $2
  i32.const 1
  i32.and
  if
   local.get $0
   local.get $4
   call $~lib/rt/tlsf/removeBlock
   local.get $1
   local.get $3
   i32.const 4
   i32.add
   local.get $2
   i32.const -4
   i32.and
   i32.add
   local.tee $3
   i32.store
   local.get $6
   local.get $1
   i32.load
   i32.const -4
   i32.and
   i32.add
   local.tee $4
   i32.load
   local.set $2
  end
  local.get $3
  i32.const 2
  i32.and
  if
   local.get $1
   i32.const 4
   i32.sub
   i32.load
   local.tee $1
   i32.load
   local.set $6
   local.get $0
   local.get $1
   call $~lib/rt/tlsf/removeBlock
   local.get $1
   local.get $6
   i32.const 4
   i32.add
   local.get $3
   i32.const -4
   i32.and
   i32.add
   local.tee $3
   i32.store
  end
  local.get $4
  local.get $2
  i32.const 2
  i32.or
  i32.store
  local.get $4
  i32.const 4
  i32.sub
  local.get $1
  i32.store
  local.get $0
  local.get $3
  i32.const -4
  i32.and
  local.tee $2
  i32.const 256
  i32.lt_u
  if (result i32)
   local.get $2
   i32.const 4
   i32.shr_u
  else
   i32.const 31
   i32.const 1073741820
   local.get $2
   local.get $2
   i32.const 1073741820
   i32.ge_u
   select
   local.tee $2
   i32.clz
   i32.sub
   local.tee $3
   i32.const 7
   i32.sub
   local.set $5
   local.get $2
   local.get $3
   i32.const 4
   i32.sub
   i32.shr_u
   i32.const 16
   i32.xor
  end
  local.tee $2
  local.get $5
  i32.const 4
  i32.shl
  i32.add
  i32.const 2
  i32.shl
  i32.add
  i32.load offset=96
  local.set $3
  local.get $1
  i32.const 0
  i32.store offset=4
  local.get $1
  local.get $3
  i32.store offset=8
  local.get $3
  if
   local.get $3
   local.get $1
   i32.store offset=4
  end
  local.get $0
  local.get $5
  i32.const 4
  i32.shl
  local.get $2
  i32.add
  i32.const 2
  i32.shl
  i32.add
  local.get $1
  i32.store offset=96
  local.get $0
  local.get $0
  i32.load
  i32.const 1
  local.get $5
  i32.shl
  i32.or
  i32.store
  local.get $0
  local.get $5
  i32.const 2
  i32.shl
  i32.add
  local.tee $0
  local.get $0
  i32.load offset=4
  i32.const 1
  local.get $2
  i32.shl
  i32.or
  i32.store offset=4
 )
 (func $~lib/rt/tlsf/addMemory (param $0 i32) (param $1 i32) (param $2 i64)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  local.get $1
  i32.const 19
  i32.add
  i32.const -16
  i32.and
  i32.const 4
  i32.sub
  local.set $1
  local.get $0
  i32.load offset=1568
  local.tee $3
  if
   local.get $1
   i32.const 16
   i32.sub
   local.tee $5
   local.get $3
   i32.eq
   if
    local.get $3
    i32.load
    local.set $4
    local.get $5
    local.set $1
   end
  end
  local.get $2
  i32.wrap_i64
  i32.const -16
  i32.and
  local.get $1
  i32.sub
  local.tee $3
  i32.const 20
  i32.lt_u
  if
   return
  end
  local.get $1
  local.get $4
  i32.const 2
  i32.and
  local.get $3
  i32.const 8
  i32.sub
  local.tee $3
  i32.const 1
  i32.or
  i32.or
  i32.store
  local.get $1
  i32.const 0
  i32.store offset=4
  local.get $1
  i32.const 0
  i32.store offset=8
  local.get $1
  i32.const 4
  i32.add
  local.get $3
  i32.add
  local.tee $3
  i32.const 2
  i32.store
  local.get $0
  local.get $3
  i32.store offset=1568
  local.get $0
  local.get $1
  call $~lib/rt/tlsf/insertBlock
 )
 (func $~lib/rt/tlsf/initialize
  (local $0 i32)
  (local $1 i32)
  memory.size
  local.tee $0
  i32.const 0
  i32.le_s
  if (result i32)
   i32.const 1
   local.get $0
   i32.sub
   memory.grow
   i32.const 0
   i32.lt_s
  else
   i32.const 0
  end
  if
   unreachable
  end
  i32.const 46768
  i32.const 0
  i32.store
  i32.const 48336
  i32.const 0
  i32.store
  loop $for-loop|0
   local.get $1
   i32.const 23
   i32.lt_u
   if
    local.get $1
    i32.const 2
    i32.shl
    i32.const 46768
    i32.add
    i32.const 0
    i32.store offset=4
    i32.const 0
    local.set $0
    loop $for-loop|1
     local.get $0
     i32.const 16
     i32.lt_u
     if
      local.get $1
      i32.const 4
      i32.shl
      local.get $0
      i32.add
      i32.const 2
      i32.shl
      i32.const 46768
      i32.add
      i32.const 0
      i32.store offset=96
      local.get $0
      i32.const 1
      i32.add
      local.set $0
      br $for-loop|1
     end
    end
    local.get $1
    i32.const 1
    i32.add
    local.set $1
    br $for-loop|0
   end
  end
  i32.const 46768
  i32.const 48340
  memory.size
  i64.extend_i32_s
  i64.const 16
  i64.shl
  call $~lib/rt/tlsf/addMemory
  i32.const 46768
  global.set $~lib/rt/tlsf/ROOT
 )
 (func $~lib/rt/itcms/step (result i32)
  (local $0 i32)
  (local $1 i32)
  (local $2 i32)
  block $break|0
   block $case2|0
    block $case1|0
     block $case0|0
      global.get $~lib/rt/itcms/state
      br_table $case0|0 $case1|0 $case2|0 $break|0
     end
     i32.const 1
     global.set $~lib/rt/itcms/state
     i32.const 0
     global.set $~lib/rt/itcms/visitCount
     call $~lib/rt/itcms/visitRoots
     global.get $~lib/rt/itcms/toSpace
     global.set $~lib/rt/itcms/iter
     global.get $~lib/rt/itcms/visitCount
     return
    end
    global.get $~lib/rt/itcms/white
    i32.eqz
    local.set $1
    global.get $~lib/rt/itcms/iter
    i32.load offset=4
    i32.const -4
    i32.and
    local.set $0
    loop $while-continue|1
     local.get $0
     global.get $~lib/rt/itcms/toSpace
     i32.ne
     if
      local.get $0
      global.set $~lib/rt/itcms/iter
      local.get $1
      local.get $0
      i32.load offset=4
      local.tee $2
      i32.const 3
      i32.and
      i32.ne
      if
       local.get $0
       local.get $2
       i32.const -4
       i32.and
       local.get $1
       i32.or
       i32.store offset=4
       i32.const 0
       global.set $~lib/rt/itcms/visitCount
       local.get $0
       i32.const 20
       i32.add
       call $~lib/rt/__visit_members
       global.get $~lib/rt/itcms/visitCount
       return
      end
      local.get $0
      i32.load offset=4
      i32.const -4
      i32.and
      local.set $0
      br $while-continue|1
     end
    end
    i32.const 0
    global.set $~lib/rt/itcms/visitCount
    call $~lib/rt/itcms/visitRoots
    global.get $~lib/rt/itcms/toSpace
    global.get $~lib/rt/itcms/iter
    i32.load offset=4
    i32.const -4
    i32.and
    i32.eq
    if
     global.get $~lib/memory/__stack_pointer
     local.set $0
     loop $while-continue|0
      local.get $0
      i32.const 46760
      i32.lt_u
      if
       local.get $0
       i32.load
       call $~lib/rt/itcms/__visit
       local.get $0
       i32.const 4
       i32.add
       local.set $0
       br $while-continue|0
      end
     end
     global.get $~lib/rt/itcms/iter
     i32.load offset=4
     i32.const -4
     i32.and
     local.set $0
     loop $while-continue|2
      local.get $0
      global.get $~lib/rt/itcms/toSpace
      i32.ne
      if
       local.get $1
       local.get $0
       i32.load offset=4
       local.tee $2
       i32.const 3
       i32.and
       i32.ne
       if
        local.get $0
        local.get $2
        i32.const -4
        i32.and
        local.get $1
        i32.or
        i32.store offset=4
        local.get $0
        i32.const 20
        i32.add
        call $~lib/rt/__visit_members
       end
       local.get $0
       i32.load offset=4
       i32.const -4
       i32.and
       local.set $0
       br $while-continue|2
      end
     end
     global.get $~lib/rt/itcms/fromSpace
     local.set $0
     global.get $~lib/rt/itcms/toSpace
     global.set $~lib/rt/itcms/fromSpace
     local.get $0
     global.set $~lib/rt/itcms/toSpace
     local.get $1
     global.set $~lib/rt/itcms/white
     local.get $0
     i32.load offset=4
     i32.const -4
     i32.and
     global.set $~lib/rt/itcms/iter
     i32.const 2
     global.set $~lib/rt/itcms/state
    end
    global.get $~lib/rt/itcms/visitCount
    return
   end
   global.get $~lib/rt/itcms/iter
   local.tee $0
   global.get $~lib/rt/itcms/toSpace
   i32.ne
   if
    local.get $0
    i32.load offset=4
    i32.const -4
    i32.and
    global.set $~lib/rt/itcms/iter
    local.get $0
    i32.const 46760
    i32.lt_u
    if
     local.get $0
     i32.const 0
     i32.store offset=4
     local.get $0
     i32.const 0
     i32.store offset=8
    else
     global.get $~lib/rt/itcms/total
     local.get $0
     i32.load
     i32.const -4
     i32.and
     i32.const 4
     i32.add
     i32.sub
     global.set $~lib/rt/itcms/total
     local.get $0
     i32.const 4
     i32.add
     local.tee $0
     i32.const 46760
     i32.ge_u
     if
      global.get $~lib/rt/tlsf/ROOT
      i32.eqz
      if
       call $~lib/rt/tlsf/initialize
      end
      local.get $0
      i32.const 4
      i32.sub
      local.set $1
      local.get $0
      i32.const 15
      i32.and
      i32.const 1
      local.get $0
      select
      if (result i32)
       i32.const 1
      else
       local.get $1
       i32.load
       i32.const 1
       i32.and
      end
      drop
      local.get $1
      local.get $1
      i32.load
      i32.const 1
      i32.or
      i32.store
      global.get $~lib/rt/tlsf/ROOT
      local.get $1
      call $~lib/rt/tlsf/insertBlock
     end
    end
    i32.const 10
    return
   end
   global.get $~lib/rt/itcms/toSpace
   global.get $~lib/rt/itcms/toSpace
   i32.store offset=4
   global.get $~lib/rt/itcms/toSpace
   global.get $~lib/rt/itcms/toSpace
   i32.store offset=8
   i32.const 0
   global.set $~lib/rt/itcms/state
  end
  i32.const 0
 )
 (func $~lib/rt/tlsf/searchBlock (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  local.get $1
  i32.const 256
  i32.lt_u
  if
   local.get $1
   i32.const 4
   i32.shr_u
   local.set $1
  else
   local.get $1
   i32.const 536870910
   i32.lt_u
   if
    local.get $1
    i32.const 1
    i32.const 27
    local.get $1
    i32.clz
    i32.sub
    i32.shl
    i32.add
    i32.const 1
    i32.sub
    local.set $1
   end
   local.get $1
   i32.const 31
   local.get $1
   i32.clz
   i32.sub
   local.tee $2
   i32.const 4
   i32.sub
   i32.shr_u
   i32.const 16
   i32.xor
   local.set $1
   local.get $2
   i32.const 7
   i32.sub
   local.set $2
  end
  local.get $0
  local.get $2
  i32.const 2
  i32.shl
  i32.add
  i32.load offset=4
  i32.const -1
  local.get $1
  i32.shl
  i32.and
  local.tee $1
  if (result i32)
   local.get $0
   local.get $1
   i32.ctz
   local.get $2
   i32.const 4
   i32.shl
   i32.add
   i32.const 2
   i32.shl
   i32.add
   i32.load offset=96
  else
   local.get $0
   i32.load
   i32.const -1
   local.get $2
   i32.const 1
   i32.add
   i32.shl
   i32.and
   local.tee $1
   if (result i32)
    local.get $0
    local.get $0
    local.get $1
    i32.ctz
    local.tee $0
    i32.const 2
    i32.shl
    i32.add
    i32.load offset=4
    i32.ctz
    local.get $0
    i32.const 4
    i32.shl
    i32.add
    i32.const 2
    i32.shl
    i32.add
    i32.load offset=96
   else
    i32.const 0
   end
  end
 )
 (func $~lib/rt/itcms/__new (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  local.get $0
  i32.const 1073741804
  i32.ge_u
  if
   i32.const 1088
   i32.const 1152
   i32.const 261
   i32.const 31
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/rt/itcms/total
  global.get $~lib/rt/itcms/threshold
  i32.ge_u
  if
   block $__inlined_func$~lib/rt/itcms/interrupt$69
    i32.const 2048
    local.set $2
    loop $do-loop|0
     local.get $2
     call $~lib/rt/itcms/step
     i32.sub
     local.set $2
     global.get $~lib/rt/itcms/state
     i32.eqz
     if
      global.get $~lib/rt/itcms/total
      i64.extend_i32_u
      i64.const 200
      i64.mul
      i64.const 100
      i64.div_u
      i32.wrap_i64
      i32.const 1024
      i32.add
      global.set $~lib/rt/itcms/threshold
      br $__inlined_func$~lib/rt/itcms/interrupt$69
     end
     local.get $2
     i32.const 0
     i32.gt_s
     br_if $do-loop|0
    end
    global.get $~lib/rt/itcms/total
    global.get $~lib/rt/itcms/total
    global.get $~lib/rt/itcms/threshold
    i32.sub
    i32.const 1024
    i32.lt_u
    i32.const 10
    i32.shl
    i32.add
    global.set $~lib/rt/itcms/threshold
   end
  end
  global.get $~lib/rt/tlsf/ROOT
  i32.eqz
  if
   call $~lib/rt/tlsf/initialize
  end
  global.get $~lib/rt/tlsf/ROOT
  local.set $4
  local.get $0
  i32.const 16
  i32.add
  local.tee $2
  i32.const 1073741820
  i32.gt_u
  if
   i32.const 1088
   i32.const 1424
   i32.const 461
   i32.const 29
   call $~lib/builtins/abort
   unreachable
  end
  local.get $4
  local.get $2
  i32.const 12
  i32.le_u
  if (result i32)
   i32.const 12
  else
   local.get $2
   i32.const 19
   i32.add
   i32.const -16
   i32.and
   i32.const 4
   i32.sub
  end
  local.tee $5
  call $~lib/rt/tlsf/searchBlock
  local.tee $2
  i32.eqz
  if
   memory.size
   local.tee $2
   local.get $5
   i32.const 256
   i32.ge_u
   if (result i32)
    local.get $5
    i32.const 536870910
    i32.lt_u
    if (result i32)
     local.get $5
     i32.const 1
     i32.const 27
     local.get $5
     i32.clz
     i32.sub
     i32.shl
     i32.add
     i32.const 1
     i32.sub
    else
     local.get $5
    end
   else
    local.get $5
   end
   i32.const 4
   local.get $4
   i32.load offset=1568
   local.get $2
   i32.const 16
   i32.shl
   i32.const 4
   i32.sub
   i32.ne
   i32.shl
   i32.add
   i32.const 65535
   i32.add
   i32.const -65536
   i32.and
   i32.const 16
   i32.shr_u
   local.tee $3
   local.get $2
   local.get $3
   i32.gt_s
   select
   memory.grow
   i32.const 0
   i32.lt_s
   if
    local.get $3
    memory.grow
    i32.const 0
    i32.lt_s
    if
     unreachable
    end
   end
   local.get $4
   local.get $2
   i32.const 16
   i32.shl
   memory.size
   i64.extend_i32_s
   i64.const 16
   i64.shl
   call $~lib/rt/tlsf/addMemory
   local.get $4
   local.get $5
   call $~lib/rt/tlsf/searchBlock
   local.set $2
  end
  local.get $2
  i32.load
  drop
  local.get $4
  local.get $2
  call $~lib/rt/tlsf/removeBlock
  local.get $2
  i32.load
  local.tee $3
  i32.const -4
  i32.and
  local.get $5
  i32.sub
  local.tee $6
  i32.const 16
  i32.ge_u
  if
   local.get $2
   local.get $5
   local.get $3
   i32.const 2
   i32.and
   i32.or
   i32.store
   local.get $2
   i32.const 4
   i32.add
   local.get $5
   i32.add
   local.tee $3
   local.get $6
   i32.const 4
   i32.sub
   i32.const 1
   i32.or
   i32.store
   local.get $4
   local.get $3
   call $~lib/rt/tlsf/insertBlock
  else
   local.get $2
   local.get $3
   i32.const -2
   i32.and
   i32.store
   local.get $2
   i32.const 4
   i32.add
   local.get $2
   i32.load
   i32.const -4
   i32.and
   i32.add
   local.tee $3
   local.get $3
   i32.load
   i32.const -3
   i32.and
   i32.store
  end
  local.get $2
  local.get $1
  i32.store offset=12
  local.get $2
  local.get $0
  i32.store offset=16
  global.get $~lib/rt/itcms/fromSpace
  local.tee $1
  i32.load offset=8
  local.set $3
  local.get $2
  local.get $1
  global.get $~lib/rt/itcms/white
  i32.or
  i32.store offset=4
  local.get $2
  local.get $3
  i32.store offset=8
  local.get $3
  local.get $2
  local.get $3
  i32.load offset=4
  i32.const 3
  i32.and
  i32.or
  i32.store offset=4
  local.get $1
  local.get $2
  i32.store offset=8
  global.get $~lib/rt/itcms/total
  local.get $2
  i32.load
  i32.const -4
  i32.and
  i32.const 4
  i32.add
  i32.add
  global.set $~lib/rt/itcms/total
  local.get $2
  i32.const 20
  i32.add
  local.tee $1
  i32.const 0
  local.get $0
  memory.fill
  local.get $1
 )
 (func $~lib/rt/itcms/__link (param $0 i32) (param $1 i32) (param $2 i32)
  (local $3 i32)
  local.get $1
  i32.eqz
  if
   return
  end
  global.get $~lib/rt/itcms/white
  local.get $1
  i32.const 20
  i32.sub
  local.tee $1
  i32.load offset=4
  i32.const 3
  i32.and
  i32.eq
  if
   local.get $0
   i32.const 20
   i32.sub
   local.tee $0
   i32.load offset=4
   i32.const 3
   i32.and
   local.tee $3
   global.get $~lib/rt/itcms/white
   i32.eqz
   i32.eq
   if
    local.get $0
    local.get $1
    local.get $2
    select
    call $~lib/rt/itcms/Object#makeGray
   else
    global.get $~lib/rt/itcms/state
    i32.const 1
    i32.eq
    local.get $3
    i32.const 3
    i32.eq
    i32.and
    if
     local.get $1
     call $~lib/rt/itcms/Object#makeGray
    end
   end
  end
 )
 (func $~lib/util/number/genDigits (param $0 i64) (param $1 i64) (param $2 i32) (param $3 i64) (param $4 i32) (result i32)
  (local $5 i32)
  (local $6 i64)
  (local $7 i32)
  (local $8 i64)
  (local $9 i64)
  (local $10 i32)
  (local $11 i64)
  local.get $1
  local.get $0
  i64.sub
  local.set $8
  i64.const 1
  i32.const 0
  local.get $2
  i32.sub
  local.tee $10
  i64.extend_i32_s
  local.tee $0
  i64.shl
  local.tee $9
  i64.const 1
  i64.sub
  local.tee $11
  local.get $1
  i64.and
  local.set $6
  local.get $1
  local.get $0
  i64.shr_u
  i32.wrap_i64
  local.tee $2
  i32.const 100000
  i32.lt_u
  if (result i32)
   local.get $2
   i32.const 100
   i32.lt_u
   if (result i32)
    local.get $2
    i32.const 10
    i32.ge_u
    i32.const 1
    i32.add
   else
    local.get $2
    i32.const 10000
    i32.ge_u
    i32.const 3
    i32.add
    local.get $2
    i32.const 1000
    i32.ge_u
    i32.add
   end
  else
   local.get $2
   i32.const 10000000
   i32.lt_u
   if (result i32)
    local.get $2
    i32.const 1000000
    i32.ge_u
    i32.const 6
    i32.add
   else
    local.get $2
    i32.const 1000000000
    i32.ge_u
    i32.const 8
    i32.add
    local.get $2
    i32.const 100000000
    i32.ge_u
    i32.add
   end
  end
  local.set $7
  loop $while-continue|0
   local.get $7
   i32.const 0
   i32.gt_s
   if
    block $break|1
     block $case10|1
      block $case9|1
       block $case8|1
        block $case7|1
         block $case6|1
          block $case5|1
           block $case4|1
            block $case3|1
             block $case2|1
              block $case1|1
               block $case0|1
                local.get $7
                i32.const 1
                i32.sub
                br_table $case9|1 $case8|1 $case7|1 $case6|1 $case5|1 $case4|1 $case3|1 $case2|1 $case1|1 $case0|1 $case10|1
               end
               local.get $2
               i32.const 1000000000
               i32.div_u
               local.set $5
               local.get $2
               i32.const 1000000000
               i32.rem_u
               local.set $2
               br $break|1
              end
              local.get $2
              i32.const 100000000
              i32.div_u
              local.set $5
              local.get $2
              i32.const 100000000
              i32.rem_u
              local.set $2
              br $break|1
             end
             local.get $2
             i32.const 10000000
             i32.div_u
             local.set $5
             local.get $2
             i32.const 10000000
             i32.rem_u
             local.set $2
             br $break|1
            end
            local.get $2
            i32.const 1000000
            i32.div_u
            local.set $5
            local.get $2
            i32.const 1000000
            i32.rem_u
            local.set $2
            br $break|1
           end
           local.get $2
           i32.const 100000
           i32.div_u
           local.set $5
           local.get $2
           i32.const 100000
           i32.rem_u
           local.set $2
           br $break|1
          end
          local.get $2
          i32.const 10000
          i32.div_u
          local.set $5
          local.get $2
          i32.const 10000
          i32.rem_u
          local.set $2
          br $break|1
         end
         local.get $2
         i32.const 1000
         i32.div_u
         local.set $5
         local.get $2
         i32.const 1000
         i32.rem_u
         local.set $2
         br $break|1
        end
        local.get $2
        i32.const 100
        i32.div_u
        local.set $5
        local.get $2
        i32.const 100
        i32.rem_u
        local.set $2
        br $break|1
       end
       local.get $2
       i32.const 10
       i32.div_u
       local.set $5
       local.get $2
       i32.const 10
       i32.rem_u
       local.set $2
       br $break|1
      end
      local.get $2
      local.set $5
      i32.const 0
      local.set $2
      br $break|1
     end
     i32.const 0
     local.set $5
    end
    local.get $4
    local.get $5
    i32.or
    if
     local.get $4
     i32.const 1
     i32.shl
     i32.const 3632
     i32.add
     local.get $5
     i32.const 65535
     i32.and
     i32.const 48
     i32.add
     i32.store16
     local.get $4
     i32.const 1
     i32.add
     local.set $4
    end
    local.get $7
    i32.const 1
    i32.sub
    local.set $7
    local.get $3
    local.get $2
    i64.extend_i32_u
    local.get $10
    i64.extend_i32_s
    local.tee $1
    i64.shl
    local.get $6
    i64.add
    local.tee $0
    i64.ge_u
    if
     global.get $~lib/util/number/_K
     local.get $7
     i32.add
     global.set $~lib/util/number/_K
     local.get $7
     i32.const 2
     i32.shl
     i32.const 4560
     i32.add
     i64.load32_u
     local.get $1
     i64.shl
     local.set $1
     local.get $4
     i32.const 1
     i32.shl
     i32.const 3630
     i32.add
     local.tee $2
     i32.load16_u
     local.set $5
     loop $while-continue|3
      local.get $0
      local.get $8
      i64.lt_u
      local.get $3
      local.get $0
      i64.sub
      local.get $1
      i64.ge_u
      i32.and
      if (result i32)
       local.get $0
       local.get $1
       i64.add
       local.tee $6
       local.get $8
       i64.lt_u
       local.get $8
       local.get $0
       i64.sub
       local.get $6
       local.get $8
       i64.sub
       i64.gt_u
       i32.or
      else
       i32.const 0
      end
      if
       local.get $5
       i32.const 1
       i32.sub
       local.set $5
       local.get $0
       local.get $1
       i64.add
       local.set $0
       br $while-continue|3
      end
     end
     local.get $2
     local.get $5
     i32.store16
     local.get $4
     return
    end
    br $while-continue|0
   end
  end
  loop $while-continue|4
   local.get $3
   i64.const 10
   i64.mul
   local.set $3
   local.get $6
   i64.const 10
   i64.mul
   local.tee $0
   local.get $10
   i64.extend_i32_s
   i64.shr_u
   local.tee $1
   local.get $4
   i64.extend_i32_s
   i64.or
   i64.const 0
   i64.ne
   if
    local.get $4
    i32.const 1
    i32.shl
    i32.const 3632
    i32.add
    local.get $1
    i32.wrap_i64
    i32.const 65535
    i32.and
    i32.const 48
    i32.add
    i32.store16
    local.get $4
    i32.const 1
    i32.add
    local.set $4
   end
   local.get $7
   i32.const 1
   i32.sub
   local.set $7
   local.get $0
   local.get $11
   i64.and
   local.tee $6
   local.get $3
   i64.ge_u
   br_if $while-continue|4
  end
  global.get $~lib/util/number/_K
  local.get $7
  i32.add
  global.set $~lib/util/number/_K
  local.get $8
  i32.const 0
  local.get $7
  i32.sub
  i32.const 2
  i32.shl
  i32.const 4560
  i32.add
  i64.load32_u
  i64.mul
  local.set $0
  local.get $4
  i32.const 1
  i32.shl
  i32.const 3630
  i32.add
  local.tee $2
  i32.load16_u
  local.set $5
  loop $while-continue|6
   local.get $0
   local.get $6
   i64.gt_u
   local.get $3
   local.get $6
   i64.sub
   local.get $9
   i64.ge_u
   i32.and
   if (result i32)
    local.get $6
    local.get $9
    i64.add
    local.tee $1
    local.get $0
    i64.lt_u
    local.get $0
    local.get $6
    i64.sub
    local.get $1
    local.get $0
    i64.sub
    i64.gt_u
    i32.or
   else
    i32.const 0
   end
   if
    local.get $5
    i32.const 1
    i32.sub
    local.set $5
    local.get $6
    local.get $9
    i64.add
    local.set $6
    br $while-continue|6
   end
  end
  local.get $2
  local.get $5
  i32.store16
  local.get $4
 )
 (func $~lib/util/number/utoa32_dec_lut (param $0 i32) (param $1 i32) (param $2 i32)
  (local $3 i32)
  loop $while-continue|0
   local.get $1
   i32.const 10000
   i32.ge_u
   if
    local.get $1
    i32.const 10000
    i32.rem_u
    local.set $3
    local.get $1
    i32.const 10000
    i32.div_u
    local.set $1
    local.get $0
    local.get $2
    i32.const 4
    i32.sub
    local.tee $2
    i32.const 1
    i32.shl
    i32.add
    local.get $3
    i32.const 100
    i32.div_u
    i32.const 2
    i32.shl
    i32.const 4600
    i32.add
    i64.load32_u
    local.get $3
    i32.const 100
    i32.rem_u
    i32.const 2
    i32.shl
    i32.const 4600
    i32.add
    i64.load32_u
    i64.const 32
    i64.shl
    i64.or
    i64.store
    br $while-continue|0
   end
  end
  local.get $1
  i32.const 100
  i32.ge_u
  if
   local.get $0
   local.get $2
   i32.const 2
   i32.sub
   local.tee $2
   i32.const 1
   i32.shl
   i32.add
   local.get $1
   i32.const 100
   i32.rem_u
   i32.const 2
   i32.shl
   i32.const 4600
   i32.add
   i32.load
   i32.store
   local.get $1
   i32.const 100
   i32.div_u
   local.set $1
  end
  local.get $1
  i32.const 10
  i32.ge_u
  if
   local.get $0
   local.get $2
   i32.const 2
   i32.sub
   i32.const 1
   i32.shl
   i32.add
   local.get $1
   i32.const 2
   i32.shl
   i32.const 4600
   i32.add
   i32.load
   i32.store
  else
   local.get $0
   local.get $2
   i32.const 1
   i32.sub
   i32.const 1
   i32.shl
   i32.add
   local.get $1
   i32.const 48
   i32.add
   i32.store16
  end
 )
 (func $~lib/util/number/prettify (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  local.get $2
  i32.eqz
  if
   local.get $0
   local.get $1
   i32.const 1
   i32.shl
   i32.add
   i32.const 3145774
   i32.store
   local.get $1
   i32.const 2
   i32.add
   return
  end
  local.get $1
  local.get $2
  i32.add
  local.tee $3
  i32.const 21
  i32.le_s
  local.get $1
  local.get $3
  i32.le_s
  i32.and
  if (result i32)
   loop $for-loop|0
    local.get $1
    local.get $3
    i32.lt_s
    if
     local.get $0
     local.get $1
     i32.const 1
     i32.shl
     i32.add
     i32.const 48
     i32.store16
     local.get $1
     i32.const 1
     i32.add
     local.set $1
     br $for-loop|0
    end
   end
   local.get $0
   local.get $3
   i32.const 1
   i32.shl
   i32.add
   i32.const 3145774
   i32.store
   local.get $3
   i32.const 2
   i32.add
  else
   local.get $3
   i32.const 21
   i32.le_s
   local.get $3
   i32.const 0
   i32.gt_s
   i32.and
   if (result i32)
    local.get $0
    local.get $3
    i32.const 1
    i32.shl
    i32.add
    local.tee $0
    i32.const 2
    i32.add
    local.get $0
    i32.const 0
    local.get $2
    i32.sub
    i32.const 1
    i32.shl
    memory.copy
    local.get $0
    i32.const 46
    i32.store16
    local.get $1
    i32.const 1
    i32.add
   else
    local.get $3
    i32.const 0
    i32.le_s
    local.get $3
    i32.const -6
    i32.gt_s
    i32.and
    if (result i32)
     local.get $0
     i32.const 2
     local.get $3
     i32.sub
     local.tee $3
     i32.const 1
     i32.shl
     i32.add
     local.get $0
     local.get $1
     i32.const 1
     i32.shl
     memory.copy
     local.get $0
     i32.const 3014704
     i32.store
     i32.const 2
     local.set $2
     loop $for-loop|1
      local.get $2
      local.get $3
      i32.lt_s
      if
       local.get $0
       local.get $2
       i32.const 1
       i32.shl
       i32.add
       i32.const 48
       i32.store16
       local.get $2
       i32.const 1
       i32.add
       local.set $2
       br $for-loop|1
      end
     end
     local.get $1
     local.get $3
     i32.add
    else
     local.get $1
     i32.const 1
     i32.eq
     if
      local.get $0
      i32.const 101
      i32.store16 offset=2
      local.get $0
      i32.const 4
      i32.add
      local.tee $2
      local.get $3
      i32.const 1
      i32.sub
      local.tee $0
      i32.const 0
      i32.lt_s
      local.tee $3
      if
       i32.const 0
       local.get $0
       i32.sub
       local.set $0
      end
      local.get $0
      local.get $0
      i32.const 100000
      i32.lt_u
      if (result i32)
       local.get $0
       i32.const 100
       i32.lt_u
       if (result i32)
        local.get $0
        i32.const 10
        i32.ge_u
        i32.const 1
        i32.add
       else
        local.get $0
        i32.const 10000
        i32.ge_u
        i32.const 3
        i32.add
        local.get $0
        i32.const 1000
        i32.ge_u
        i32.add
       end
      else
       local.get $0
       i32.const 10000000
       i32.lt_u
       if (result i32)
        local.get $0
        i32.const 1000000
        i32.ge_u
        i32.const 6
        i32.add
       else
        local.get $0
        i32.const 1000000000
        i32.ge_u
        i32.const 8
        i32.add
        local.get $0
        i32.const 100000000
        i32.ge_u
        i32.add
       end
      end
      i32.const 1
      i32.add
      local.tee $1
      call $~lib/util/number/utoa32_dec_lut
      local.get $2
      i32.const 45
      i32.const 43
      local.get $3
      select
      i32.store16
     else
      local.get $0
      i32.const 4
      i32.add
      local.get $0
      i32.const 2
      i32.add
      local.get $1
      i32.const 1
      i32.shl
      local.tee $2
      i32.const 2
      i32.sub
      memory.copy
      local.get $0
      i32.const 46
      i32.store16 offset=2
      local.get $0
      local.get $2
      i32.add
      local.tee $0
      i32.const 101
      i32.store16 offset=2
      local.get $0
      i32.const 4
      i32.add
      local.tee $2
      local.get $3
      i32.const 1
      i32.sub
      local.tee $0
      i32.const 0
      i32.lt_s
      local.tee $3
      if
       i32.const 0
       local.get $0
       i32.sub
       local.set $0
      end
      local.get $0
      local.get $0
      i32.const 100000
      i32.lt_u
      if (result i32)
       local.get $0
       i32.const 100
       i32.lt_u
       if (result i32)
        local.get $0
        i32.const 10
        i32.ge_u
        i32.const 1
        i32.add
       else
        local.get $0
        i32.const 10000
        i32.ge_u
        i32.const 3
        i32.add
        local.get $0
        i32.const 1000
        i32.ge_u
        i32.add
       end
      else
       local.get $0
       i32.const 10000000
       i32.lt_u
       if (result i32)
        local.get $0
        i32.const 1000000
        i32.ge_u
        i32.const 6
        i32.add
       else
        local.get $0
        i32.const 1000000000
        i32.ge_u
        i32.const 8
        i32.add
        local.get $0
        i32.const 100000000
        i32.ge_u
        i32.add
       end
      end
      i32.const 1
      i32.add
      local.tee $0
      call $~lib/util/number/utoa32_dec_lut
      local.get $2
      i32.const 45
      i32.const 43
      local.get $3
      select
      i32.store16
      local.get $0
      local.get $1
      i32.add
      local.set $1
     end
     local.get $1
     i32.const 2
     i32.add
    end
   end
  end
 )
 (func $~lib/util/number/dtoa_core (param $0 f64) (result i32)
  (local $1 i64)
  (local $2 i32)
  (local $3 i64)
  (local $4 i64)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i64)
  (local $9 i64)
  (local $10 i64)
  (local $11 i64)
  (local $12 i64)
  (local $13 i64)
  (local $14 i64)
  local.get $0
  f64.const 0
  f64.lt
  local.tee $2
  if (result f64)
   i32.const 3632
   i32.const 45
   i32.store16
   local.get $0
   f64.neg
  else
   local.get $0
  end
  i64.reinterpret_f64
  local.tee $1
  i64.const 9218868437227405312
  i64.and
  i64.const 52
  i64.shr_u
  i32.wrap_i64
  local.tee $5
  i32.const 1
  local.get $5
  select
  i32.const 1075
  i32.sub
  local.tee $6
  i32.const 1
  i32.sub
  local.get $1
  i64.const 4503599627370495
  i64.and
  local.get $5
  i32.const 0
  i32.ne
  i64.extend_i32_u
  i64.const 52
  i64.shl
  i64.add
  local.tee $1
  i64.const 1
  i64.shl
  i64.const 1
  i64.add
  local.tee $3
  i64.clz
  i32.wrap_i64
  local.tee $5
  i32.sub
  local.set $7
  local.get $3
  local.get $5
  i64.extend_i32_s
  i64.shl
  global.set $~lib/util/number/_frc_plus
  local.get $1
  local.get $1
  i64.const 4503599627370496
  i64.eq
  i32.const 1
  i32.add
  local.tee $5
  i64.extend_i32_s
  i64.shl
  i64.const 1
  i64.sub
  local.get $6
  local.get $5
  i32.sub
  local.get $7
  i32.sub
  i64.extend_i32_s
  i64.shl
  global.set $~lib/util/number/_frc_minus
  local.get $7
  global.set $~lib/util/number/_exp
  i32.const 348
  i32.const -61
  global.get $~lib/util/number/_exp
  i32.sub
  f64.convert_i32_s
  f64.const 0.30102999566398114
  f64.mul
  f64.const 347
  f64.add
  local.tee $0
  i32.trunc_sat_f64_s
  local.tee $5
  local.get $5
  f64.convert_i32_s
  local.get $0
  f64.ne
  i32.add
  i32.const 3
  i32.shr_s
  i32.const 1
  i32.add
  local.tee $5
  i32.const 3
  i32.shl
  local.tee $6
  i32.sub
  global.set $~lib/util/number/_K
  local.get $6
  i32.const 3688
  i32.add
  i64.load
  global.set $~lib/util/number/_frc_pow
  local.get $5
  i32.const 1
  i32.shl
  i32.const 4384
  i32.add
  i32.load16_s
  global.set $~lib/util/number/_exp_pow
  local.get $1
  local.get $1
  i64.clz
  i64.shl
  local.tee $1
  i64.const 4294967295
  i64.and
  local.set $8
  global.get $~lib/util/number/_frc_pow
  local.tee $9
  i64.const 4294967295
  i64.and
  local.tee $10
  local.get $1
  i64.const 32
  i64.shr_u
  local.tee $3
  i64.mul
  local.get $8
  local.get $10
  i64.mul
  i64.const 32
  i64.shr_u
  i64.add
  local.set $11
  global.get $~lib/util/number/_frc_plus
  local.tee $1
  i64.const 4294967295
  i64.and
  local.set $12
  local.get $1
  i64.const 32
  i64.shr_u
  local.tee $4
  local.get $10
  i64.mul
  local.get $10
  local.get $12
  i64.mul
  i64.const 32
  i64.shr_u
  i64.add
  local.set $13
  global.get $~lib/util/number/_frc_minus
  local.tee $14
  i64.const 4294967295
  i64.and
  local.set $1
  local.get $14
  i64.const 32
  i64.shr_u
  local.tee $14
  local.get $10
  i64.mul
  local.get $1
  local.get $10
  i64.mul
  i64.const 32
  i64.shr_u
  i64.add
  local.set $10
  local.get $4
  local.get $9
  i64.const 32
  i64.shr_u
  local.tee $4
  i64.mul
  local.get $13
  i64.const 32
  i64.shr_u
  i64.add
  local.get $4
  local.get $12
  i64.mul
  local.get $13
  i64.const 4294967295
  i64.and
  i64.add
  i64.const 2147483647
  i64.add
  i64.const 32
  i64.shr_u
  i64.add
  i64.const 1
  i64.sub
  local.set $9
  local.get $2
  i32.const 1
  i32.shl
  i32.const 3632
  i32.add
  local.get $3
  local.get $4
  i64.mul
  local.get $11
  i64.const 32
  i64.shr_u
  i64.add
  local.get $4
  local.get $8
  i64.mul
  local.get $11
  i64.const 4294967295
  i64.and
  i64.add
  i64.const 2147483647
  i64.add
  i64.const 32
  i64.shr_u
  i64.add
  local.get $9
  global.get $~lib/util/number/_exp_pow
  global.get $~lib/util/number/_exp
  i32.add
  i32.const -64
  i32.sub
  local.get $9
  local.get $4
  local.get $14
  i64.mul
  local.get $10
  i64.const 32
  i64.shr_u
  i64.add
  local.get $1
  local.get $4
  i64.mul
  local.get $10
  i64.const 4294967295
  i64.and
  i64.add
  i64.const 2147483647
  i64.add
  i64.const 32
  i64.shr_u
  i64.add
  i64.const 1
  i64.add
  i64.sub
  local.get $2
  call $~lib/util/number/genDigits
  local.get $2
  i32.sub
  global.get $~lib/util/number/_K
  call $~lib/util/number/prettify
  local.get $2
  i32.add
 )
 (func $~lib/number/F64#toString (param $0 f64) (result i32)
  (local $1 i32)
  (local $2 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  i32.const 3488
  local.set $1
  block $~lib/util/number/dtoa_impl|inlined.0
   local.get $0
   f64.const 0
   f64.eq
   br_if $~lib/util/number/dtoa_impl|inlined.0
   local.get $0
   local.get $0
   f64.sub
   f64.const 0
   f64.ne
   if
    i32.const 3520
    local.set $1
    local.get $0
    local.get $0
    f64.ne
    br_if $~lib/util/number/dtoa_impl|inlined.0
    i32.const 3552
    i32.const 3600
    local.get $0
    f64.const 0
    f64.lt
    select
    local.set $1
    br $~lib/util/number/dtoa_impl|inlined.0
   end
   local.get $0
   call $~lib/util/number/dtoa_core
   i32.const 1
   i32.shl
   local.set $2
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.const 2
   call $~lib/rt/itcms/__new
   local.tee $1
   i32.store
   local.get $1
   i32.const 3632
   local.get $2
   memory.copy
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $1
 )
 (func $~lib/util/string/compareImpl (param $0 i32) (param $1 i32) (param $2 i32) (param $3 i32) (result i32)
  (local $4 i32)
  local.get $0
  local.get $1
  i32.const 1
  i32.shl
  i32.add
  local.set $1
  local.get $3
  i32.const 4
  i32.ge_u
  if (result i32)
   local.get $1
   i32.const 7
   i32.and
   local.get $2
   i32.const 7
   i32.and
   i32.or
  else
   i32.const 1
  end
  i32.eqz
  if
   loop $do-loop|0
    local.get $1
    i64.load
    local.get $2
    i64.load
    i64.eq
    if
     local.get $1
     i32.const 8
     i32.add
     local.set $1
     local.get $2
     i32.const 8
     i32.add
     local.set $2
     local.get $3
     i32.const 4
     i32.sub
     local.tee $3
     i32.const 4
     i32.ge_u
     br_if $do-loop|0
    end
   end
  end
  loop $while-continue|1
   local.get $3
   local.tee $0
   i32.const 1
   i32.sub
   local.set $3
   local.get $0
   if
    local.get $1
    i32.load16_u
    local.tee $0
    local.get $2
    i32.load16_u
    local.tee $4
    i32.ne
    if
     local.get $0
     local.get $4
     i32.sub
     return
    end
    local.get $1
    i32.const 2
    i32.add
    local.set $1
    local.get $2
    i32.const 2
    i32.add
    local.set $2
    br $while-continue|1
   end
  end
  i32.const 0
 )
 (func $~lib/number/I32#toString (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  block $__inlined_func$~lib/util/number/itoa32$83
   local.get $0
   i32.eqz
   if
    global.get $~lib/memory/__stack_pointer
    i32.const 4
    i32.add
    global.set $~lib/memory/__stack_pointer
    i32.const 9904
    local.set $0
    br $__inlined_func$~lib/util/number/itoa32$83
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   local.get $0
   i32.sub
   local.get $0
   local.get $0
   i32.const 31
   i32.shr_u
   i32.const 1
   i32.shl
   local.tee $2
   select
   local.tee $1
   i32.const 100000
   i32.lt_u
   if (result i32)
    local.get $1
    i32.const 100
    i32.lt_u
    if (result i32)
     local.get $1
     i32.const 10
     i32.ge_u
     i32.const 1
     i32.add
    else
     local.get $1
     i32.const 10000
     i32.ge_u
     i32.const 3
     i32.add
     local.get $1
     i32.const 1000
     i32.ge_u
     i32.add
    end
   else
    local.get $1
    i32.const 10000000
    i32.lt_u
    if (result i32)
     local.get $1
     i32.const 1000000
     i32.ge_u
     i32.const 6
     i32.add
    else
     local.get $1
     i32.const 1000000000
     i32.ge_u
     i32.const 8
     i32.add
     local.get $1
     i32.const 100000000
     i32.ge_u
     i32.add
    end
   end
   local.tee $3
   i32.const 1
   i32.shl
   local.get $2
   i32.add
   i32.const 2
   call $~lib/rt/itcms/__new
   local.tee $0
   i32.store
   local.get $0
   local.get $2
   i32.add
   local.get $1
   local.get $3
   call $~lib/util/number/utoa32_dec_lut
   local.get $2
   if
    local.get $0
    i32.const 45
    i32.store16
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
  end
  local.get $0
 )
 (func $assembly/index/getWasmVersion (result i32)
  i32.const 11552
 )
 (func $assembly/index/getSupportedFeatures (result i32)
  i32.const 11584
 )
 (func $assembly/index/deallocateString (param $0 f64)
 )
 (func $~lib/rt/itcms/__pin (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  local.get $0
  if
   local.get $0
   i32.const 20
   i32.sub
   local.tee $1
   i32.load offset=4
   i32.const 3
   i32.and
   i32.const 3
   i32.eq
   if
    i32.const 13840
    i32.const 1152
    i32.const 338
    i32.const 7
    call $~lib/builtins/abort
    unreachable
   end
   block $__inlined_func$~lib/rt/itcms/Object#unlink$2
    local.get $1
    i32.load offset=4
    i32.const -4
    i32.and
    local.tee $2
    i32.eqz
    if
     local.get $1
     i32.load offset=8
     drop
     br $__inlined_func$~lib/rt/itcms/Object#unlink$2
    end
    local.get $2
    local.get $1
    i32.load offset=8
    local.tee $3
    i32.store offset=8
    local.get $3
    local.get $2
    local.get $3
    i32.load offset=4
    i32.const 3
    i32.and
    i32.or
    i32.store offset=4
   end
   global.get $~lib/rt/itcms/pinSpace
   local.tee $2
   i32.load offset=8
   local.set $3
   local.get $1
   local.get $2
   i32.const 3
   i32.or
   i32.store offset=4
   local.get $1
   local.get $3
   i32.store offset=8
   local.get $3
   local.get $1
   local.get $3
   i32.load offset=4
   i32.const 3
   i32.and
   i32.or
   i32.store offset=4
   local.get $2
   local.get $1
   i32.store offset=8
  end
  local.get $0
 )
 (func $~lib/rt/itcms/__unpin (param $0 i32)
  (local $1 i32)
  (local $2 i32)
  local.get $0
  i32.eqz
  if
   return
  end
  local.get $0
  i32.const 20
  i32.sub
  local.tee $0
  i32.load offset=4
  i32.const 3
  i32.and
  i32.const 3
  i32.ne
  if
   i32.const 13904
   i32.const 1152
   i32.const 352
   i32.const 5
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/rt/itcms/state
  i32.const 1
  i32.eq
  if
   local.get $0
   call $~lib/rt/itcms/Object#makeGray
  else
   block $__inlined_func$~lib/rt/itcms/Object#unlink$3
    local.get $0
    i32.load offset=4
    i32.const -4
    i32.and
    local.tee $1
    i32.eqz
    if
     local.get $0
     i32.load offset=8
     drop
     br $__inlined_func$~lib/rt/itcms/Object#unlink$3
    end
    local.get $1
    local.get $0
    i32.load offset=8
    local.tee $2
    i32.store offset=8
    local.get $2
    local.get $1
    local.get $2
    i32.load offset=4
    i32.const 3
    i32.and
    i32.or
    i32.store offset=4
   end
   global.get $~lib/rt/itcms/fromSpace
   local.tee $1
   i32.load offset=8
   local.set $2
   local.get $0
   local.get $1
   global.get $~lib/rt/itcms/white
   i32.or
   i32.store offset=4
   local.get $0
   local.get $2
   i32.store offset=8
   local.get $2
   local.get $0
   local.get $2
   i32.load offset=4
   i32.const 3
   i32.and
   i32.or
   i32.store offset=4
   local.get $1
   local.get $0
   i32.store offset=8
  end
 )
 (func $~lib/rt/itcms/__collect
  global.get $~lib/rt/itcms/state
  i32.const 0
  i32.gt_s
  if
   loop $while-continue|0
    global.get $~lib/rt/itcms/state
    if
     call $~lib/rt/itcms/step
     drop
     br $while-continue|0
    end
   end
  end
  call $~lib/rt/itcms/step
  drop
  loop $while-continue|1
   global.get $~lib/rt/itcms/state
   if
    call $~lib/rt/itcms/step
    drop
    br $while-continue|1
   end
  end
  global.get $~lib/rt/itcms/total
  i64.extend_i32_u
  i64.const 200
  i64.mul
  i64.const 100
  i64.div_u
  i32.wrap_i64
  i32.const 1024
  i32.add
  global.set $~lib/rt/itcms/threshold
 )
 (func $~lib/rt/__visit_members (param $0 i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  block $folding-inner2
   block $folding-inner1
    block $folding-inner0
     block $invalid
      block $~lib/function/Function<%28~lib/string/String%2Ci32%2C~lib/array/Array<~lib/string/String>%29=>~lib/string/String>
       block $~lib/array/Array<i32>
        block $~lib/array/Array<~lib/string/String>
         block $~lib/staticarray/StaticArray<~lib/string/String>
          block $"~lib/map/Map<~lib/string/String,~lib/string/String>"
           block $~lib/arraybuffer/ArrayBufferView
            block $~lib/string/String
             block $~lib/arraybuffer/ArrayBuffer
              block $~lib/object/Object
               local.get $0
               i32.const 8
               i32.sub
               i32.load
               br_table $~lib/object/Object $~lib/arraybuffer/ArrayBuffer $~lib/string/String $~lib/arraybuffer/ArrayBufferView $"~lib/map/Map<~lib/string/String,~lib/string/String>" $~lib/staticarray/StaticArray<~lib/string/String> $~lib/array/Array<~lib/string/String> $~lib/array/Array<i32> $~lib/function/Function<%28~lib/string/String%2Ci32%2C~lib/array/Array<~lib/string/String>%29=>~lib/string/String> $invalid
              end
              return
             end
             return
            end
            return
           end
           local.get $0
           i32.load
           local.tee $0
           if
            local.get $0
            call $~lib/rt/itcms/__visit
           end
           return
          end
          global.get $~lib/memory/__stack_pointer
          i32.const 4
          i32.sub
          global.set $~lib/memory/__stack_pointer
          global.get $~lib/memory/__stack_pointer
          i32.const 13992
          i32.lt_s
          br_if $folding-inner0
          global.get $~lib/memory/__stack_pointer
          i32.const 0
          i32.store
          global.get $~lib/memory/__stack_pointer
          local.get $0
          i32.store
          local.get $0
          i32.load
          call $~lib/rt/itcms/__visit
          global.get $~lib/memory/__stack_pointer
          local.get $0
          i32.store
          local.get $0
          i32.load offset=8
          local.tee $2
          local.set $1
          global.get $~lib/memory/__stack_pointer
          local.get $0
          i32.store
          local.get $1
          local.get $0
          i32.load offset=16
          i32.const 12
          i32.mul
          i32.add
          local.set $0
          loop $while-continue|0
           local.get $0
           local.get $1
           i32.gt_u
           if
            local.get $1
            i32.load offset=8
            i32.const 1
            i32.and
            i32.eqz
            if
             local.get $1
             i32.load
             call $~lib/rt/itcms/__visit
             local.get $1
             i32.load offset=4
             call $~lib/rt/itcms/__visit
            end
            local.get $1
            i32.const 12
            i32.add
            local.set $1
            br $while-continue|0
           end
          end
          local.get $2
          call $~lib/rt/itcms/__visit
          br $folding-inner2
         end
         local.get $0
         local.get $0
         i32.const 20
         i32.sub
         i32.load offset=16
         i32.add
         local.set $1
         loop $while-continue|01
          local.get $0
          local.get $1
          i32.lt_u
          if
           local.get $0
           i32.load
           local.tee $2
           if
            local.get $2
            call $~lib/rt/itcms/__visit
           end
           local.get $0
           i32.const 4
           i32.add
           local.set $0
           br $while-continue|01
          end
         end
         return
        end
        global.get $~lib/memory/__stack_pointer
        i32.const 4
        i32.sub
        global.set $~lib/memory/__stack_pointer
        global.get $~lib/memory/__stack_pointer
        i32.const 13992
        i32.lt_s
        br_if $folding-inner0
        global.get $~lib/memory/__stack_pointer
        i32.const 0
        i32.store
        global.get $~lib/memory/__stack_pointer
        local.get $0
        i32.store
        local.get $0
        i32.load offset=4
        local.set $1
        global.get $~lib/memory/__stack_pointer
        local.get $0
        i32.store
        local.get $1
        local.get $0
        i32.load offset=12
        i32.const 2
        i32.shl
        i32.add
        local.set $2
        loop $while-continue|02
         local.get $1
         local.get $2
         i32.lt_u
         if
          local.get $1
          i32.load
          local.tee $3
          if
           local.get $3
           call $~lib/rt/itcms/__visit
          end
          local.get $1
          i32.const 4
          i32.add
          local.set $1
          br $while-continue|02
         end
        end
        br $folding-inner1
       end
       global.get $~lib/memory/__stack_pointer
       i32.const 4
       i32.sub
       global.set $~lib/memory/__stack_pointer
       global.get $~lib/memory/__stack_pointer
       i32.const 13992
       i32.lt_s
       br_if $folding-inner0
       global.get $~lib/memory/__stack_pointer
       i32.const 0
       i32.store
       br $folding-inner1
      end
      global.get $~lib/memory/__stack_pointer
      i32.const 4
      i32.sub
      global.set $~lib/memory/__stack_pointer
      global.get $~lib/memory/__stack_pointer
      i32.const 13992
      i32.lt_s
      br_if $folding-inner0
      global.get $~lib/memory/__stack_pointer
      i32.const 0
      i32.store
      global.get $~lib/memory/__stack_pointer
      local.get $0
      i32.store
      local.get $0
      i32.load offset=4
      call $~lib/rt/itcms/__visit
      br $folding-inner2
     end
     unreachable
    end
    i32.const 46784
    i32.const 46832
    i32.const 1
    i32.const 1
    call $~lib/builtins/abort
    unreachable
   end
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   i32.load
   call $~lib/rt/itcms/__visit
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
 )
 (func $~setArgumentsLength (param $0 i32)
  local.get $0
  global.set $~argumentsLength
 )
 (func $~start
  memory.size
  i32.const 16
  i32.shl
  i32.const 46760
  i32.sub
  i32.const 1
  i32.shr_u
  global.set $~lib/rt/itcms/threshold
  i32.const 1204
  i32.const 1200
  i32.store
  i32.const 1208
  i32.const 1200
  i32.store
  i32.const 1200
  global.set $~lib/rt/itcms/pinSpace
  i32.const 1236
  i32.const 1232
  i32.store
  i32.const 1240
  i32.const 1232
  i32.store
  i32.const 1232
  global.set $~lib/rt/itcms/toSpace
  i32.const 1380
  i32.const 1376
  i32.store
  i32.const 1384
  i32.const 1376
  i32.store
  i32.const 1376
  global.set $~lib/rt/itcms/fromSpace
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#constructor"
  global.set $assembly/index/apiKeys
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#constructor"
  global.set $assembly/index/providerEndpoints
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#constructor"
  global.set $assembly/index/providerModels
 )
 (func $"~lib/map/Map<~lib/string/String,~lib/string/String>#constructor" (result i32)
  (local $0 i32)
  (local $1 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  i32.const 24
  i32.const 4
  call $~lib/rt/itcms/__new
  local.tee $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  i32.const 16
  call $~lib/arraybuffer/ArrayBuffer#constructor
  local.set $1
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=8
  local.get $0
  local.get $1
  i32.store
  local.get $0
  local.get $1
  i32.const 0
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  local.get $0
  i32.const 3
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  i32.const 48
  call $~lib/arraybuffer/ArrayBuffer#constructor
  local.set $1
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=8
  local.get $0
  local.get $1
  i32.store offset=8
  local.get $0
  local.get $1
  i32.const 0
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  local.get $0
  i32.const 4
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  local.get $0
  i32.const 0
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  local.get $0
  i32.const 0
  i32.store offset=20
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $~lib/string/String#concat (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const -2
  i32.and
  local.set $2
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store
  local.get $1
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const -2
  i32.and
  local.tee $3
  local.get $2
  i32.add
  local.tee $4
  i32.eqz
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1056
   return
  end
  global.get $~lib/memory/__stack_pointer
  local.get $4
  i32.const 2
  call $~lib/rt/itcms/__new
  local.tee $4
  i32.store offset=4
  local.get $4
  local.get $0
  local.get $2
  memory.copy
  local.get $2
  local.get $4
  i32.add
  local.get $1
  local.get $3
  memory.copy
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $4
 )
 (func $~lib/console/console.log (param $0 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  call $~lib/bindings/dom/console.log
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
 )
 (func $assembly/index/initializeAgent (result i32)
  (local $0 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  i32.const 1680
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/agentId
  local.tee $0
  i32.store offset=8
  i32.const 1680
  local.get $0
  call $~lib/string/String#concat
  local.set $0
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  call $~lib/console/console.log
  i32.const 1
  global.set $assembly/index/agentInitialized
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.add
  global.set $~lib/memory/__stack_pointer
  i32.const 1
 )
 (func $~lib/util/string/joinStringArray (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store offset=8
  local.get $1
  i32.const 1
  i32.sub
  local.tee $5
  i32.const 0
  i32.lt_s
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1056
   return
  end
  local.get $5
  i32.eqz
  if
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.load
   local.tee $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   i32.const 1056
   local.get $0
   select
   return
  end
  loop $for-loop|0
   local.get $1
   local.get $4
   i32.gt_s
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    local.get $4
    i32.const 2
    i32.shl
    i32.add
    i32.load
    local.tee $6
    i32.store offset=4
    local.get $6
    if
     global.get $~lib/memory/__stack_pointer
     local.get $6
     i32.store offset=8
     local.get $3
     local.get $6
     i32.const 20
     i32.sub
     i32.load offset=16
     i32.const 1
     i32.shr_u
     i32.add
     local.set $3
    end
    local.get $4
    i32.const 1
    i32.add
    local.set $4
    br $for-loop|0
   end
  end
  i32.const 0
  local.set $4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $3
  local.get $2
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 1
  i32.shr_u
  local.tee $1
  local.get $5
  i32.mul
  i32.add
  i32.const 1
  i32.shl
  i32.const 2
  call $~lib/rt/itcms/__new
  local.tee $6
  i32.store offset=12
  i32.const 0
  local.set $3
  loop $for-loop|1
   local.get $3
   local.get $5
   i32.lt_s
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    local.get $3
    i32.const 2
    i32.shl
    i32.add
    i32.load
    local.tee $7
    i32.store offset=4
    local.get $7
    if
     global.get $~lib/memory/__stack_pointer
     local.get $7
     i32.store offset=8
     local.get $6
     local.get $4
     i32.const 1
     i32.shl
     i32.add
     local.get $7
     local.get $7
     i32.const 20
     i32.sub
     i32.load offset=16
     i32.const 1
     i32.shr_u
     local.tee $7
     i32.const 1
     i32.shl
     memory.copy
     local.get $4
     local.get $7
     i32.add
     local.set $4
    end
    local.get $1
    if
     local.get $6
     local.get $4
     i32.const 1
     i32.shl
     i32.add
     local.get $2
     local.get $1
     i32.const 1
     i32.shl
     memory.copy
     local.get $1
     local.get $4
     i32.add
     local.set $4
    end
    local.get $3
    i32.const 1
    i32.add
    local.set $3
    br $for-loop|1
   end
  end
  global.get $~lib/memory/__stack_pointer
  local.get $0
  local.get $5
  i32.const 2
  i32.shl
  i32.add
  i32.load
  local.tee $0
  i32.store offset=4
  local.get $0
  if
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   local.get $6
   local.get $4
   i32.const 1
   i32.shl
   i32.add
   local.get $0
   local.get $0
   i32.const 20
   i32.sub
   i32.load offset=16
   i32.const -2
   i32.and
   memory.copy
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $6
 )
 (func $~lib/staticarray/StaticArray<~lib/string/String>#join (param $0 i32) (result i32)
  (local $1 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  local.get $0
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 2
  i32.shr_u
  local.set $1
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store
  local.get $0
  local.get $1
  i32.const 1056
  call $~lib/util/string/joinStringArray
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/getAgentStatus (result i32)
  (local $0 i32)
  (local $1 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store offset=8
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/agentId
  local.tee $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 3088
  i32.const 3120
  global.get $assembly/index/agentInitialized
  select
  local.tee $1
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 3040
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=12
  i32.const 3044
  local.get $0
  i32.store
  i32.const 3040
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 3040
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=12
  i32.const 3052
  local.get $1
  i32.store
  i32.const 3040
  local.get $1
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 3040
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=12
  i32.const 3040
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/loadModelWeights (param $0 f64) (param $1 f64) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 24
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.const 24
  memory.fill
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/modelType
  local.tee $3
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  call $~lib/number/F64#toString
  local.tee $4
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $1
  call $~lib/number/F64#toString
  local.tee $2
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  i32.const 3440
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=20
  i32.const 3444
  local.get $3
  i32.store
  i32.const 3440
  local.get $3
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 3440
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  local.get $4
  i32.store offset=20
  i32.const 3452
  local.get $4
  i32.store
  i32.const 3440
  local.get $4
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 3440
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=20
  i32.const 3460
  local.get $2
  i32.store
  i32.const 3440
  local.get $2
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 3440
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=20
  i32.const 3440
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $2
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store
  local.get $2
  call $~lib/console/console.log
  i32.const 1
  global.set $assembly/index/modelLoaded
  global.get $~lib/memory/__stack_pointer
  i32.const 24
  i32.add
  global.set $~lib/memory/__stack_pointer
  i32.const 1
 )
 (func $assembly/index/getModelInfo (result i32)
  (local $0 i32)
  (local $1 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store offset=8
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/modelType
  local.tee $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 3088
  i32.const 3120
  global.get $assembly/index/modelLoaded
  select
  local.tee $1
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 5824
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=12
  i32.const 5828
  local.get $0
  i32.store
  i32.const 5824
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 5824
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=12
  i32.const 5836
  local.get $1
  i32.store
  i32.const 5824
  local.get $1
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 5824
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=12
  i32.const 5824
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $~lib/util/hash/HASH<~lib/string/String> (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  block $~lib/util/hash/hashStr|inlined.0 (result i32)
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   i32.const 0
   local.get $0
   i32.eqz
   br_if $~lib/util/hash/hashStr|inlined.0
   drop
   global.get $~lib/memory/__stack_pointer
   local.get $0
   local.tee $1
   i32.store offset=4
   local.get $1
   i32.const 20
   i32.sub
   i32.load offset=16
   i32.const -2
   i32.and
   local.tee $3
   i32.const 16
   i32.ge_u
   if (result i32)
    i32.const 606290984
    local.set $2
    i32.const -2048144777
    local.set $4
    i32.const 1640531535
    local.set $5
    local.get $1
    local.get $3
    i32.add
    i32.const 16
    i32.sub
    local.set $7
    loop $while-continue|0
     local.get $1
     local.get $7
     i32.le_u
     if
      local.get $2
      local.get $1
      i32.load
      i32.const -2048144777
      i32.mul
      i32.add
      i32.const 13
      i32.rotl
      i32.const -1640531535
      i32.mul
      local.set $2
      local.get $4
      local.get $1
      i32.load offset=4
      i32.const -2048144777
      i32.mul
      i32.add
      i32.const 13
      i32.rotl
      i32.const -1640531535
      i32.mul
      local.set $4
      local.get $6
      local.get $1
      i32.load offset=8
      i32.const -2048144777
      i32.mul
      i32.add
      i32.const 13
      i32.rotl
      i32.const -1640531535
      i32.mul
      local.set $6
      local.get $5
      local.get $1
      i32.load offset=12
      i32.const -2048144777
      i32.mul
      i32.add
      i32.const 13
      i32.rotl
      i32.const -1640531535
      i32.mul
      local.set $5
      local.get $1
      i32.const 16
      i32.add
      local.set $1
      br $while-continue|0
     end
    end
    local.get $3
    local.get $2
    i32.const 1
    i32.rotl
    local.get $4
    i32.const 7
    i32.rotl
    i32.add
    local.get $6
    i32.const 12
    i32.rotl
    i32.add
    local.get $5
    i32.const 18
    i32.rotl
    i32.add
    i32.add
   else
    local.get $3
    i32.const 374761393
    i32.add
   end
   local.set $2
   local.get $0
   local.get $3
   i32.add
   i32.const 4
   i32.sub
   local.set $4
   loop $while-continue|1
    local.get $1
    local.get $4
    i32.le_u
    if
     local.get $2
     local.get $1
     i32.load
     i32.const -1028477379
     i32.mul
     i32.add
     i32.const 17
     i32.rotl
     i32.const 668265263
     i32.mul
     local.set $2
     local.get $1
     i32.const 4
     i32.add
     local.set $1
     br $while-continue|1
    end
   end
   local.get $0
   local.get $3
   i32.add
   local.set $0
   loop $while-continue|2
    local.get $0
    local.get $1
    i32.gt_u
    if
     local.get $2
     local.get $1
     i32.load8_u
     i32.const 374761393
     i32.mul
     i32.add
     i32.const 11
     i32.rotl
     i32.const -1640531535
     i32.mul
     local.set $2
     local.get $1
     i32.const 1
     i32.add
     local.set $1
     br $while-continue|2
    end
   end
   local.get $2
   local.get $2
   i32.const 15
   i32.shr_u
   i32.xor
   i32.const -2048144777
   i32.mul
   local.tee $0
   local.get $0
   i32.const 13
   i32.shr_u
   i32.xor
   i32.const -1028477379
   i32.mul
   local.tee $0
   local.get $0
   i32.const 16
   i32.shr_u
   i32.xor
  end
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $~lib/string/String.__eq (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  local.get $0
  local.get $1
  i32.eq
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1
   return
  end
  block $folding-inner0
   local.get $1
   i32.eqz
   local.get $0
   i32.eqz
   i32.or
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   i32.const 20
   i32.sub
   i32.load offset=16
   i32.const 1
   i32.shr_u
   local.set $2
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store
   local.get $2
   local.get $1
   i32.const 20
   i32.sub
   i32.load offset=16
   i32.const 1
   i32.shr_u
   i32.ne
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   local.get $0
   i32.const 0
   local.get $1
   local.get $2
   call $~lib/util/string/compareImpl
   i32.eqz
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  i32.const 0
 )
 (func $"~lib/map/Map<~lib/string/String,~lib/string/String>#find" (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  i32.load
  local.set $3
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $3
  local.get $2
  local.get $0
  i32.load offset=4
  i32.and
  i32.const 2
  i32.shl
  i32.add
  i32.load
  local.set $2
  loop $while-continue|0
   local.get $2
   if
    local.get $2
    i32.load offset=8
    local.tee $0
    i32.const 1
    i32.and
    if (result i32)
     i32.const 0
    else
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.load
     local.tee $3
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=4
     local.get $3
     local.get $1
     call $~lib/string/String.__eq
    end
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 8
     i32.add
     global.set $~lib/memory/__stack_pointer
     local.get $2
     return
    end
    local.get $0
    i32.const -2
    i32.and
    local.set $2
    br $while-continue|0
   end
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  i32.const 0
 )
 (func $"~lib/map/Map<~lib/string/String,~lib/string/String>#set" (param $0 i32) (param $1 i32) (param $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i32)
  (local $9 i32)
  (local $10 i32)
  (local $11 i32)
  (local $12 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store
   local.get $1
   call $~lib/util/hash/HASH<~lib/string/String>
   local.set $8
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   local.get $0
   local.get $1
   local.get $8
   call $"~lib/map/Map<~lib/string/String,~lib/string/String>#find"
   local.tee $3
   if
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store
    local.get $3
    local.get $2
    i32.store offset=4
    local.get $0
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
   else
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    local.get $0
    i32.load offset=16
    local.set $3
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    local.get $3
    local.get $0
    i32.load offset=12
    i32.eq
    if
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=4
     local.get $0
     i32.load offset=20
     local.set $3
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=4
     local.get $3
     local.get $0
     i32.load offset=12
     i32.const 3
     i32.mul
     i32.const 4
     i32.div_s
     i32.lt_s
     if (result i32)
      global.get $~lib/memory/__stack_pointer
      local.get $0
      i32.store offset=4
      local.get $0
      i32.load offset=4
     else
      global.get $~lib/memory/__stack_pointer
      local.get $0
      i32.store offset=4
      local.get $0
      i32.load offset=4
      i32.const 1
      i32.shl
      i32.const 1
      i32.or
     end
     local.set $9
     global.get $~lib/memory/__stack_pointer
     i32.const 20
     i32.sub
     global.set $~lib/memory/__stack_pointer
     global.get $~lib/memory/__stack_pointer
     i32.const 13992
     i32.lt_s
     br_if $folding-inner0
     global.get $~lib/memory/__stack_pointer
     i32.const 0
     i32.const 20
     memory.fill
     global.get $~lib/memory/__stack_pointer
     local.get $9
     i32.const 1
     i32.add
     local.tee $3
     i32.const 2
     i32.shl
     call $~lib/arraybuffer/ArrayBuffer#constructor
     local.tee $10
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $3
     i32.const 3
     i32.shl
     i32.const 3
     i32.div_s
     local.tee $7
     i32.const 12
     i32.mul
     call $~lib/arraybuffer/ArrayBuffer#constructor
     local.tee $4
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=8
     local.get $0
     i32.load offset=8
     local.set $11
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=8
     local.get $11
     local.get $0
     i32.load offset=16
     i32.const 12
     i32.mul
     i32.add
     local.set $6
     local.get $4
     local.set $3
     loop $while-continue|0
      local.get $6
      local.get $11
      i32.ne
      if
       local.get $11
       i32.load offset=8
       i32.const 1
       i32.and
       i32.eqz
       if
        global.get $~lib/memory/__stack_pointer
        local.get $11
        i32.load
        local.tee $12
        i32.store offset=12
        global.get $~lib/memory/__stack_pointer
        local.get $12
        i32.store offset=8
        local.get $3
        local.get $12
        i32.store
        global.get $~lib/memory/__stack_pointer
        local.get $11
        i32.load offset=4
        local.tee $5
        i32.store offset=8
        local.get $3
        local.get $5
        i32.store offset=4
        global.get $~lib/memory/__stack_pointer
        local.get $12
        i32.store offset=8
        local.get $3
        local.get $10
        local.get $12
        call $~lib/util/hash/HASH<~lib/string/String>
        local.get $9
        i32.and
        i32.const 2
        i32.shl
        i32.add
        local.tee $5
        i32.load
        i32.store offset=8
        local.get $5
        local.get $3
        i32.store
        local.get $3
        i32.const 12
        i32.add
        local.set $3
       end
       local.get $11
       i32.const 12
       i32.add
       local.set $11
       br $while-continue|0
      end
     end
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=8
     global.get $~lib/memory/__stack_pointer
     local.get $10
     i32.store offset=16
     local.get $0
     local.get $10
     i32.store
     local.get $0
     local.get $10
     i32.const 0
     call $~lib/rt/itcms/__link
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=8
     local.get $0
     local.get $9
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=8
     global.get $~lib/memory/__stack_pointer
     local.get $4
     i32.store offset=16
     local.get $0
     local.get $4
     i32.store offset=8
     local.get $0
     local.get $4
     i32.const 0
     call $~lib/rt/itcms/__link
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=8
     local.get $0
     local.get $7
     i32.store offset=12
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=8
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=16
     local.get $0
     local.get $0
     i32.load offset=20
     i32.store offset=16
     global.get $~lib/memory/__stack_pointer
     i32.const 20
     i32.add
     global.set $~lib/memory/__stack_pointer
    end
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.load offset=8
    local.tee $3
    i32.store offset=8
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    local.get $0
    local.get $0
    i32.load offset=16
    local.tee $4
    i32.const 1
    i32.add
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store
    local.get $3
    local.get $4
    i32.const 12
    i32.mul
    i32.add
    local.tee $3
    local.get $1
    i32.store
    local.get $0
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store
    local.get $3
    local.get $2
    i32.store offset=4
    local.get $0
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    local.get $0
    local.get $0
    i32.load offset=20
    i32.const 1
    i32.add
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    local.get $0
    i32.load
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    local.get $3
    local.get $1
    local.get $8
    local.get $0
    i32.load offset=4
    i32.and
    i32.const 2
    i32.shl
    i32.add
    local.tee $0
    i32.load
    i32.store offset=8
    local.get $0
    local.get $3
    i32.store
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $assembly/index/configureExternalInference (param $0 i32) (param $1 i32) (param $2 i32) (param $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 24
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.const 24
  memory.fill
  global.get $~lib/memory/__stack_pointer
  i32.const 5872
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=8
  i32.const 5872
  local.get $0
  call $~lib/string/String#concat
  local.set $4
  global.get $~lib/memory/__stack_pointer
  local.get $4
  i32.store
  local.get $4
  call $~lib/console/console.log
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/apiKeys
  local.tee $4
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=8
  local.get $4
  local.get $0
  local.get $1
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#set"
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/providerEndpoints
  local.tee $1
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=8
  local.get $1
  local.get $0
  local.get $2
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#set"
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/providerModels
  local.tee $1
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=8
  local.get $1
  local.get $0
  local.get $3
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#set"
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=20
  global.get $~lib/memory/__stack_pointer
  i32.const 6160
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=8
  i32.const 6164
  local.get $0
  i32.store
  i32.const 6160
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6160
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=8
  i32.const 6172
  local.get $2
  i32.store
  i32.const 6160
  local.get $2
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6160
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=8
  i32.const 6180
  local.get $3
  i32.store
  i32.const 6160
  local.get $3
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6160
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=8
  i32.const 6160
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  call $~lib/console/console.log
  global.get $~lib/memory/__stack_pointer
  i32.const 24
  i32.add
  global.set $~lib/memory/__stack_pointer
 )
 (func $"~lib/map/Map<~lib/string/String,~lib/string/String>#has" (param $0 i32) (param $1 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=8
  local.get $0
  local.get $1
  local.get $1
  call $~lib/util/hash/HASH<~lib/string/String>
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#find"
  i32.const 0
  i32.ne
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/setActiveInferenceProvider (param $0 i32) (result i32)
  (local $1 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store offset=8
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/apiKeys
  local.tee $1
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  local.get $1
  local.get $0
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
  if (result i32)
   local.get $0
   global.set $assembly/index/activeProvider
   i32.const 1
   global.set $assembly/index/externalInferenceEnabled
   global.get $~lib/memory/__stack_pointer
   i32.const 6208
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 6208
   local.get $0
   call $~lib/string/String#concat
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1
  else
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 6368
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 6372
   local.get $0
   i32.store
   i32.const 6368
   local.get $0
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 6368
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=8
   i32.const 6368
   call $~lib/staticarray/StaticArray<~lib/string/String>#join
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 0
  end
 )
 (func $~lib/array/Array<~lib/string/String>#push (param $0 i32) (param $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   i32.load offset=12
   local.tee $4
   i32.const 1
   i32.add
   local.set $5
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $5
   local.get $0
   i32.load offset=8
   local.tee $2
   i32.const 2
   i32.shr_u
   i32.gt_u
   if
    local.get $5
    i32.const 268435455
    i32.gt_u
    if
     i32.const 1488
     i32.const 6464
     i32.const 19
     i32.const 48
     call $~lib/builtins/abort
     unreachable
    end
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    block $__inlined_func$~lib/rt/itcms/__renew$282
     i32.const 1073741820
     local.get $2
     i32.const 1
     i32.shl
     local.tee $2
     local.get $2
     i32.const 1073741820
     i32.ge_u
     select
     local.tee $2
     i32.const 8
     local.get $5
     local.get $5
     i32.const 8
     i32.le_u
     select
     i32.const 2
     i32.shl
     local.tee $3
     local.get $2
     local.get $3
     i32.gt_u
     select
     local.tee $6
     local.get $0
     i32.load
     local.tee $3
     i32.const 20
     i32.sub
     local.tee $7
     i32.load
     i32.const -4
     i32.and
     i32.const 16
     i32.sub
     i32.le_u
     if
      local.get $7
      local.get $6
      i32.store offset=16
      local.get $3
      local.set $2
      br $__inlined_func$~lib/rt/itcms/__renew$282
     end
     local.get $6
     local.get $7
     i32.load offset=12
     call $~lib/rt/itcms/__new
     local.tee $2
     local.get $3
     local.get $6
     local.get $7
     i32.load offset=16
     local.tee $7
     local.get $6
     local.get $7
     i32.lt_u
     select
     memory.copy
    end
    local.get $2
    local.get $3
    i32.ne
    if
     local.get $0
     local.get $2
     i32.store
     local.get $0
     local.get $2
     i32.store offset=4
     local.get $0
     local.get $2
     i32.const 0
     call $~lib/rt/itcms/__link
    end
    local.get $0
    local.get $6
    i32.store offset=8
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   i32.load offset=4
   local.get $4
   i32.const 2
   i32.shl
   i32.add
   local.get $1
   i32.store
   local.get $0
   local.get $1
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   local.get $5
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $assembly/index/getConfiguredProviders~anonymous|0 (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 6832
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=8
  i32.const 6836
  local.get $0
  i32.store
  i32.const 6832
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6832
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=8
  i32.const 6832
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/getConfiguredProviders (result i32)
  (local $0 i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 24
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 24
   memory.fill
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 6400
   call $~lib/rt/__newArray
   local.tee $3
   i32.store
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/apiKeys
   local.tee $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 6432
   i32.store offset=8
   local.get $1
   i32.const 6432
   call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
   if
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 6432
    i32.store offset=8
    local.get $3
    i32.const 6432
    call $~lib/array/Array<~lib/string/String>#push
   end
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/apiKeys
   local.tee $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 6512
   i32.store offset=8
   local.get $1
   i32.const 6512
   call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
   if
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 6512
    i32.store offset=8
    local.get $3
    i32.const 6512
    call $~lib/array/Array<~lib/string/String>#push
   end
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/apiKeys
   local.tee $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 6560
   i32.store offset=8
   local.get $1
   i32.const 6560
   call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
   if
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 6560
    i32.store offset=8
    local.get $3
    i32.const 6560
    call $~lib/array/Array<~lib/string/String>#push
   end
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/apiKeys
   local.tee $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 6608
   i32.store offset=8
   local.get $1
   i32.const 6608
   call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
   if
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 6608
    i32.store offset=8
    local.get $3
    i32.const 6608
    call $~lib/array/Array<~lib/string/String>#push
   end
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/apiKeys
   local.tee $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 6640
   i32.store offset=8
   local.get $1
   i32.const 6640
   call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
   if
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 6640
    i32.store offset=8
    local.get $3
    i32.const 6640
    call $~lib/array/Array<~lib/string/String>#push
   end
   global.get $~lib/memory/__stack_pointer
   local.set $1
   global.get $~lib/memory/__stack_pointer
   local.get $3
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 6864
   i32.store offset=16
   global.get $~lib/memory/__stack_pointer
   i32.const 20
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 20
   memory.fill
   global.get $~lib/memory/__stack_pointer
   local.get $3
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $3
   i32.load offset=12
   local.tee $5
   i32.const 0
   call $~lib/rt/__newArray
   local.tee $4
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $4
   i32.store
   local.get $4
   i32.load offset=4
   local.set $2
   loop $for-loop|0
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store
    local.get $0
    local.get $5
    local.get $3
    i32.load offset=12
    local.tee $6
    local.get $5
    local.get $6
    i32.lt_s
    select
    i32.lt_s
    if
     global.get $~lib/memory/__stack_pointer
     local.get $3
     i32.store offset=12
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.const 2
     i32.shl
     local.tee $6
     local.get $3
     i32.load offset=4
     i32.add
     i32.load
     local.tee $7
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $3
     i32.store offset=8
     i32.const 3
     global.set $~argumentsLength
     global.get $~lib/memory/__stack_pointer
     local.get $7
     local.get $0
     local.get $3
     i32.const 6864
     i32.load
     call_indirect (type $3)
     local.tee $7
     i32.store offset=16
     local.get $2
     local.get $6
     i32.add
     local.get $7
     i32.store
     local.get $4
     local.get $7
     i32.const 1
     call $~lib/rt/itcms/__link
     local.get $0
     i32.const 1
     i32.add
     local.set $0
     br $for-loop|0
    end
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 20
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   local.get $4
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 6896
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $4
   i32.store
   local.get $4
   i32.load offset=4
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $4
   i32.store
   local.get $4
   i32.load offset=12
   local.set $2
   global.get $~lib/memory/__stack_pointer
   i32.const 6896
   i32.store
   local.get $0
   local.get $2
   i32.const 6896
   call $~lib/util/string/joinStringArray
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $1
   local.get $0
   i32.store offset=20
   global.get $~lib/memory/__stack_pointer
   i32.const 6768
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 6772
   local.get $0
   i32.store
   i32.const 6768
   local.get $0
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 6768
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=8
   i32.const 6768
   call $~lib/staticarray/StaticArray<~lib/string/String>#join
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 24
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $"~lib/map/Map<~lib/string/String,~lib/string/String>#get" (param $0 i32) (param $1 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=8
  local.get $0
  local.get $1
  local.get $1
  call $~lib/util/hash/HASH<~lib/string/String>
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#find"
  local.tee $0
  i32.eqz
  if
   i32.const 7232
   i32.const 7296
   i32.const 105
   i32.const 17
   call $~lib/builtins/abort
   unreachable
  end
  local.get $0
  i32.load offset=4
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $~lib/string/String.__ne (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=4
  local.get $0
  i32.const 1056
  call $~lib/string/String.__eq
  i32.eqz
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $~lib/string/String.__concat (param $0 i32) (param $1 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=4
  local.get $0
  local.get $1
  call $~lib/string/String#concat
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/generateSimulatedResponse (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const -64
  i32.add
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.const 64
  memory.fill
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 6432
  i32.store offset=8
  local.get $0
  i32.const 6432
  call $~lib/string/String.__eq
  if
   global.get $~lib/memory/__stack_pointer
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 7744
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=8
   i32.const 7748
   local.get $1
   i32.store
   i32.const 7744
   local.get $1
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 7744
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=8
   local.get $0
   i32.const 7744
   call $~lib/staticarray/StaticArray<~lib/string/String>#join
   local.tee $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=8
   local.get $2
   call $~lib/string/String.__ne
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    i32.const 7888
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=24
    i32.const 7892
    local.get $2
    i32.store
    i32.const 7888
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 7888
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=24
    i32.const 7888
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $2
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=8
    local.get $1
    local.get $0
    local.get $2
    call $~lib/string/String.__concat
    local.tee $0
    i32.store
   end
  else
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 6512
   i32.store offset=8
   local.get $0
   i32.const 6512
   call $~lib/string/String.__eq
   if
    global.get $~lib/memory/__stack_pointer
    local.set $0
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=28
    global.get $~lib/memory/__stack_pointer
    i32.const 8048
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=8
    i32.const 8052
    local.get $1
    i32.store
    i32.const 8048
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 8048
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=8
    local.get $0
    i32.const 8048
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.tee $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=8
    local.get $2
    call $~lib/string/String.__ne
    if
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     local.set $1
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store offset=32
     global.get $~lib/memory/__stack_pointer
     i32.const 8128
     i32.store offset=20
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store offset=24
     i32.const 8132
     local.get $2
     i32.store
     i32.const 8128
     local.get $2
     i32.const 1
     call $~lib/rt/itcms/__link
     global.get $~lib/memory/__stack_pointer
     i32.const 8128
     i32.store offset=20
     global.get $~lib/memory/__stack_pointer
     i32.const 1056
     i32.store offset=24
     i32.const 8128
     call $~lib/staticarray/StaticArray<~lib/string/String>#join
     local.set $2
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store offset=8
     local.get $1
     local.get $0
     local.get $2
     call $~lib/string/String.__concat
     local.tee $0
     i32.store
    end
   else
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 6560
    i32.store offset=8
    local.get $0
    i32.const 6560
    call $~lib/string/String.__eq
    if
     global.get $~lib/memory/__stack_pointer
     local.set $0
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=36
     global.get $~lib/memory/__stack_pointer
     i32.const 8256
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=8
     i32.const 8260
     local.get $1
     i32.store
     i32.const 8256
     local.get $1
     i32.const 1
     call $~lib/rt/itcms/__link
     global.get $~lib/memory/__stack_pointer
     i32.const 8256
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     i32.const 1056
     i32.store offset=8
     local.get $0
     i32.const 8256
     call $~lib/staticarray/StaticArray<~lib/string/String>#join
     local.tee $0
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     i32.const 1056
     i32.store offset=8
     local.get $2
     call $~lib/string/String.__ne
     if
      global.get $~lib/memory/__stack_pointer
      local.get $0
      i32.store offset=4
      global.get $~lib/memory/__stack_pointer
      local.set $1
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store offset=40
      global.get $~lib/memory/__stack_pointer
      i32.const 8336
      i32.store offset=20
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store offset=24
      i32.const 8340
      local.get $2
      i32.store
      i32.const 8336
      local.get $2
      i32.const 1
      call $~lib/rt/itcms/__link
      global.get $~lib/memory/__stack_pointer
      i32.const 8336
      i32.store offset=20
      global.get $~lib/memory/__stack_pointer
      i32.const 1056
      i32.store offset=24
      i32.const 8336
      call $~lib/staticarray/StaticArray<~lib/string/String>#join
      local.set $2
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store offset=8
      local.get $1
      local.get $0
      local.get $2
      call $~lib/string/String.__concat
      local.tee $0
      i32.store
     end
    else
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     i32.const 6608
     i32.store offset=8
     local.get $0
     i32.const 6608
     call $~lib/string/String.__eq
     if
      global.get $~lib/memory/__stack_pointer
      local.set $0
      global.get $~lib/memory/__stack_pointer
      local.get $1
      i32.store offset=44
      global.get $~lib/memory/__stack_pointer
      i32.const 8464
      i32.store offset=4
      global.get $~lib/memory/__stack_pointer
      local.get $1
      i32.store offset=8
      i32.const 8468
      local.get $1
      i32.store
      i32.const 8464
      local.get $1
      i32.const 1
      call $~lib/rt/itcms/__link
      global.get $~lib/memory/__stack_pointer
      i32.const 8464
      i32.store offset=4
      global.get $~lib/memory/__stack_pointer
      i32.const 1056
      i32.store offset=8
      local.get $0
      i32.const 8464
      call $~lib/staticarray/StaticArray<~lib/string/String>#join
      local.tee $0
      i32.store
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store offset=4
      global.get $~lib/memory/__stack_pointer
      i32.const 1056
      i32.store offset=8
      local.get $2
      call $~lib/string/String.__ne
      if
       global.get $~lib/memory/__stack_pointer
       local.get $0
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       local.set $1
       global.get $~lib/memory/__stack_pointer
       local.get $2
       i32.store offset=48
       global.get $~lib/memory/__stack_pointer
       i32.const 8576
       i32.store offset=20
       global.get $~lib/memory/__stack_pointer
       local.get $2
       i32.store offset=24
       i32.const 8580
       local.get $2
       i32.store
       i32.const 8576
       local.get $2
       i32.const 1
       call $~lib/rt/itcms/__link
       global.get $~lib/memory/__stack_pointer
       i32.const 8576
       i32.store offset=20
       global.get $~lib/memory/__stack_pointer
       i32.const 1056
       i32.store offset=24
       i32.const 8576
       call $~lib/staticarray/StaticArray<~lib/string/String>#join
       local.set $2
       global.get $~lib/memory/__stack_pointer
       local.get $2
       i32.store offset=8
       local.get $1
       local.get $0
       local.get $2
       call $~lib/string/String.__concat
       local.tee $0
       i32.store
      end
     else
      global.get $~lib/memory/__stack_pointer
      local.get $0
      i32.store offset=4
      global.get $~lib/memory/__stack_pointer
      i32.const 6640
      i32.store offset=8
      local.get $0
      i32.const 6640
      call $~lib/string/String.__eq
      if
       global.get $~lib/memory/__stack_pointer
       local.set $0
       global.get $~lib/memory/__stack_pointer
       local.get $1
       i32.store offset=52
       global.get $~lib/memory/__stack_pointer
       i32.const 8688
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       local.get $1
       i32.store offset=8
       i32.const 8692
       local.get $1
       i32.store
       i32.const 8688
       local.get $1
       i32.const 1
       call $~lib/rt/itcms/__link
       global.get $~lib/memory/__stack_pointer
       i32.const 8688
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       i32.const 1056
       i32.store offset=8
       local.get $0
       i32.const 8688
       call $~lib/staticarray/StaticArray<~lib/string/String>#join
       local.tee $0
       i32.store
       global.get $~lib/memory/__stack_pointer
       local.get $2
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       i32.const 1056
       i32.store offset=8
       local.get $2
       call $~lib/string/String.__ne
       if
        global.get $~lib/memory/__stack_pointer
        local.get $0
        i32.store offset=4
        global.get $~lib/memory/__stack_pointer
        local.set $1
        global.get $~lib/memory/__stack_pointer
        local.get $2
        i32.store offset=56
        global.get $~lib/memory/__stack_pointer
        i32.const 8720
        i32.store offset=20
        global.get $~lib/memory/__stack_pointer
        local.get $2
        i32.store offset=24
        i32.const 8724
        local.get $2
        i32.store
        i32.const 8720
        local.get $2
        i32.const 1
        call $~lib/rt/itcms/__link
        global.get $~lib/memory/__stack_pointer
        i32.const 8720
        i32.store offset=20
        global.get $~lib/memory/__stack_pointer
        i32.const 1056
        i32.store offset=24
        i32.const 8720
        call $~lib/staticarray/StaticArray<~lib/string/String>#join
        local.set $2
        global.get $~lib/memory/__stack_pointer
        local.get $2
        i32.store offset=8
        local.get $1
        local.get $0
        local.get $2
        call $~lib/string/String.__concat
        local.tee $0
        i32.store
       end
      else
       global.get $~lib/memory/__stack_pointer
       local.set $0
       global.get $~lib/memory/__stack_pointer
       local.get $1
       i32.store offset=60
       global.get $~lib/memory/__stack_pointer
       i32.const 8848
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       local.get $1
       i32.store offset=8
       i32.const 8852
       local.get $1
       i32.store
       i32.const 8848
       local.get $1
       i32.const 1
       call $~lib/rt/itcms/__link
       global.get $~lib/memory/__stack_pointer
       i32.const 8848
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       i32.const 1056
       i32.store offset=8
       local.get $0
       i32.const 8848
       call $~lib/staticarray/StaticArray<~lib/string/String>#join
       local.tee $0
       i32.store
      end
     end
    end
   end
  end
  global.get $~lib/memory/__stack_pointer
  i32.const -64
  i32.sub
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/performExternalInference (param $0 i32) (param $1 i32) (param $2 f64) (param $3 f64) (result i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  (local $8 i32)
  (local $9 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 76
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.const 76
  memory.fill
  global.get $assembly/index/externalInferenceEnabled
  if (result i32)
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/activeProvider
   local.tee $4
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=4
   local.get $4
   i32.const 1056
   call $~lib/string/String.__eq
  else
   i32.const 1
  end
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 76
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 6928
   return
  end
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/apiKeys
  local.tee $4
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $5
  i32.store offset=4
  local.get $4
  local.get $5
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
  i32.eqz
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 76
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 7088
   return
  end
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/apiKeys
  local.tee $4
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $5
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $4
  local.get $5
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#get"
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/providerEndpoints
  local.tee $4
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $5
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $4
  local.get $5
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#get"
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/providerModels
  local.tee $4
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $5
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $4
  local.get $5
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#get"
  local.tee $5
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $4
  i32.store offset=20
  global.get $~lib/memory/__stack_pointer
  local.get $5
  i32.store offset=24
  global.get $~lib/memory/__stack_pointer
  local.get $2
  call $~lib/number/F64#toString
  local.tee $6
  i32.store offset=28
  global.get $~lib/memory/__stack_pointer
  local.get $3
  call $~lib/number/F64#toString
  local.tee $7
  i32.store offset=32
  global.get $~lib/memory/__stack_pointer
  i32.const 7600
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $4
  i32.store offset=36
  i32.const 7604
  local.get $4
  i32.store
  i32.const 7600
  local.get $4
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 7600
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $5
  i32.store offset=36
  i32.const 7612
  local.get $5
  i32.store
  i32.const 7600
  local.get $5
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 7600
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $6
  i32.store offset=36
  i32.const 7620
  local.get $6
  i32.store
  i32.const 7600
  local.get $6
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 7600
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $7
  i32.store offset=36
  i32.const 7628
  local.get $7
  i32.store
  i32.const 7600
  local.get $7
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 7600
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=36
  i32.const 7600
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $4
  global.get $~lib/memory/__stack_pointer
  local.get $4
  i32.store
  local.get $4
  call $~lib/console/console.log
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $4
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=36
  global.get $~lib/memory/__stack_pointer
  local.get $4
  local.get $0
  local.get $1
  call $assembly/index/generateSimulatedResponse
  local.tee $6
  i32.store offset=40
  global.get $~lib/memory/__stack_pointer
  local.get $6
  i32.store offset=44
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $7
  i32.store offset=48
  global.get $~lib/memory/__stack_pointer
  local.get $5
  i32.store offset=52
  global.get $~lib/memory/__stack_pointer
  local.get $2
  call $~lib/number/F64#toString
  local.tee $1
  i32.store offset=56
  global.get $~lib/memory/__stack_pointer
  local.get $3
  call $~lib/number/F64#toString
  local.tee $4
  i32.store offset=60
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 3
  i32.shr_u
  call $~lib/number/I32#toString
  local.tee $8
  i32.store offset=64
  global.get $~lib/memory/__stack_pointer
  local.get $6
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $6
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 3
  i32.shr_u
  call $~lib/number/I32#toString
  local.tee $9
  i32.store offset=68
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 1
  i32.shr_u
  local.set $0
  global.get $~lib/memory/__stack_pointer
  local.get $6
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  local.get $6
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 1
  i32.shr_u
  i32.add
  i32.const 4
  i32.div_s
  call $~lib/number/I32#toString
  local.tee $0
  i32.store offset=72
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $6
  i32.store offset=4
  i32.const 9620
  local.get $6
  i32.store
  i32.const 9616
  local.get $6
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $7
  i32.store offset=4
  i32.const 9628
  local.get $7
  i32.store
  i32.const 9616
  local.get $7
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $5
  i32.store offset=4
  i32.const 9636
  local.get $5
  i32.store
  i32.const 9616
  local.get $5
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=4
  i32.const 9644
  local.get $1
  i32.store
  i32.const 9616
  local.get $1
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $4
  i32.store offset=4
  i32.const 9652
  local.get $4
  i32.store
  i32.const 9616
  local.get $4
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $8
  i32.store offset=4
  i32.const 9660
  local.get $8
  i32.store
  i32.const 9616
  local.get $8
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $9
  i32.store offset=4
  i32.const 9668
  local.get $9
  i32.store
  i32.const 9616
  local.get $9
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  i32.const 9676
  local.get $0
  i32.store
  i32.const 9616
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9616
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=4
  i32.const 9616
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 76
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/getExternalInferenceStatus (result i32)
  (local $0 i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 20
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 20
   memory.fill
   global.get $~lib/memory/__stack_pointer
   i32.const 3088
   i32.const 3120
   global.get $assembly/index/externalInferenceEnabled
   select
   local.tee $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/activeProvider
   local.tee $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.set $2
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/apiKeys
   local.tee $3
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $3
   i32.store
   local.get $3
   i32.load offset=20
   local.set $3
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $2
   local.get $3
   call $~lib/number/I32#toString
   local.tee $2
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 11504
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=16
   i32.const 11508
   local.get $0
   i32.store
   i32.const 11504
   local.get $0
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 11504
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=16
   i32.const 11516
   local.get $1
   i32.store
   i32.const 11504
   local.get $1
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 11504
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=16
   i32.const 11524
   local.get $2
   i32.store
   i32.const 11504
   local.get $2
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 11504
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=16
   i32.const 11504
   call $~lib/staticarray/StaticArray<~lib/string/String>#join
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 20
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $~lib/string/String#indexOf (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store
  local.get $1
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 1
  i32.shr_u
  local.tee $3
  i32.eqz
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 0
   return
  end
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 1
  i32.shr_u
  local.tee $4
  i32.eqz
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const -1
   return
  end
  local.get $2
  i32.const 0
  local.get $2
  i32.const 0
  i32.gt_s
  select
  local.tee $2
  local.get $4
  local.get $2
  local.get $4
  i32.lt_s
  select
  local.set $2
  local.get $4
  local.get $3
  i32.sub
  local.set $4
  loop $for-loop|0
   local.get $2
   local.get $4
   i32.le_s
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=4
    local.get $0
    local.get $2
    local.get $1
    local.get $3
    call $~lib/util/string/compareImpl
    i32.eqz
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 8
     i32.add
     global.set $~lib/memory/__stack_pointer
     local.get $2
     return
    end
    local.get $2
    i32.const 1
    i32.add
    local.set $2
    br $for-loop|0
   end
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  i32.const -1
 )
 (func $~lib/string/String#includes (param $0 i32) (param $1 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=4
  local.get $0
  local.get $1
  i32.const 0
  call $~lib/string/String#indexOf
  i32.const -1
  i32.ne
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $~lib/string/String#substring (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i64.const 0
  i64.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $1
  i32.const 0
  local.get $1
  i32.const 0
  i32.gt_s
  select
  local.tee $3
  local.get $0
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 1
  i32.shr_u
  local.tee $1
  local.get $1
  local.get $3
  i32.gt_s
  select
  local.tee $3
  local.get $2
  i32.const 0
  local.get $2
  i32.const 0
  i32.gt_s
  select
  local.tee $2
  local.get $1
  local.get $1
  local.get $2
  i32.gt_s
  select
  local.tee $2
  local.get $2
  local.get $3
  i32.gt_s
  select
  i32.const 1
  i32.shl
  local.set $4
  local.get $3
  local.get $2
  local.get $2
  local.get $3
  i32.lt_s
  select
  i32.const 1
  i32.shl
  local.tee $2
  local.get $4
  i32.sub
  local.tee $3
  i32.eqz
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1056
   return
  end
  local.get $4
  i32.eqz
  local.get $2
  local.get $1
  i32.const 1
  i32.shl
  i32.eq
  i32.and
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.const 2
  call $~lib/rt/itcms/__new
  local.tee $1
  i32.store offset=4
  local.get $1
  local.get $0
  local.get $4
  i32.add
  local.get $3
  memory.copy
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $1
 )
 (func $assembly/index/performChatCompletion (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 20
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 20
   memory.fill
   global.get $~lib/memory/__stack_pointer
   i32.const 11936
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=8
   i32.const 11936
   local.get $1
   call $~lib/string/String#concat
   local.set $1
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store
   local.get $1
   call $~lib/console/console.log
   global.get $assembly/index/externalInferenceEnabled
   if (result i32)
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/activeProvider
    local.tee $1
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=4
    local.get $1
    i32.const 1056
    call $~lib/string/String.__eq
   else
    i32.const 1
   end
   if
    global.get $~lib/memory/__stack_pointer
    i32.const 20
    i32.add
    global.set $~lib/memory/__stack_pointer
    i32.const 6928
    return
   end
   i32.const 1056
   local.set $2
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=12
   i32.const 1056
   local.set $3
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=16
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12080
   i32.store offset=4
   local.get $0
   i32.const 12080
   call $~lib/string/String#includes
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 12080
    i32.store offset=4
    i32.const 1
    global.set $~argumentsLength
    global.get $~lib/memory/__stack_pointer
    i32.const 8
    i32.sub
    global.set $~lib/memory/__stack_pointer
    global.get $~lib/memory/__stack_pointer
    i32.const 13992
    i32.lt_s
    br_if $folding-inner0
    global.get $~lib/memory/__stack_pointer
    i64.const 0
    i64.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 12080
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 8
    i32.sub
    global.set $~lib/memory/__stack_pointer
    global.get $~lib/memory/__stack_pointer
    i32.const 13992
    i32.lt_s
    br_if $folding-inner0
    global.get $~lib/memory/__stack_pointer
    i64.const 0
    i64.store
    global.get $~lib/memory/__stack_pointer
    i32.const 12080
    i32.store
    block $__inlined_func$~lib/string/String#lastIndexOf$314
     i32.const 12076
     i32.load
     i32.const 1
     i32.shr_u
     local.tee $4
     i32.eqz
     if
      global.get $~lib/memory/__stack_pointer
      local.get $0
      i32.store
      local.get $0
      i32.const 20
      i32.sub
      i32.load offset=16
      i32.const 1
      i32.shr_u
      local.set $1
      global.get $~lib/memory/__stack_pointer
      i32.const 8
      i32.add
      global.set $~lib/memory/__stack_pointer
      br $__inlined_func$~lib/string/String#lastIndexOf$314
     end
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store
     local.get $0
     i32.const 20
     i32.sub
     i32.load offset=16
     i32.const 1
     i32.shr_u
     local.tee $1
     i32.eqz
     if
      global.get $~lib/memory/__stack_pointer
      i32.const 8
      i32.add
      global.set $~lib/memory/__stack_pointer
      i32.const -1
      local.set $1
      br $__inlined_func$~lib/string/String#lastIndexOf$314
     end
     local.get $1
     local.get $4
     i32.sub
     local.set $1
     loop $for-loop|0
      local.get $1
      i32.const 0
      i32.ge_s
      if
       global.get $~lib/memory/__stack_pointer
       local.get $0
       i32.store
       global.get $~lib/memory/__stack_pointer
       i32.const 12080
       i32.store offset=4
       local.get $0
       local.get $1
       i32.const 12080
       local.get $4
       call $~lib/util/string/compareImpl
       i32.eqz
       if
        global.get $~lib/memory/__stack_pointer
        i32.const 8
        i32.add
        global.set $~lib/memory/__stack_pointer
        br $__inlined_func$~lib/string/String#lastIndexOf$314
       end
       local.get $1
       i32.const 1
       i32.sub
       local.set $1
       br $for-loop|0
      end
     end
     global.get $~lib/memory/__stack_pointer
     i32.const 8
     i32.add
     global.set $~lib/memory/__stack_pointer
     i32.const -1
     local.set $1
    end
    global.get $~lib/memory/__stack_pointer
    i32.const 8
    i32.add
    global.set $~lib/memory/__stack_pointer
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 12128
    i32.store offset=4
    local.get $0
    i32.const 12128
    local.get $1
    call $~lib/string/String#indexOf
    i32.const 11
    i32.add
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 6800
    i32.store offset=4
    local.get $0
    i32.const 6800
    local.get $1
    call $~lib/string/String#indexOf
    local.tee $4
    local.get $1
    i32.gt_s
    local.get $1
    i32.const 10
    i32.gt_s
    i32.and
    if
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $0
     local.get $1
     local.get $4
     call $~lib/string/String#substring
     local.tee $2
     i32.store offset=12
    end
   end
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12176
   i32.store offset=4
   local.get $0
   i32.const 12176
   call $~lib/string/String#includes
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 12176
    i32.store offset=4
    local.get $0
    i32.const 12176
    i32.const 0
    call $~lib/string/String#indexOf
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 12128
    i32.store offset=4
    local.get $0
    i32.const 12128
    local.get $1
    call $~lib/string/String#indexOf
    i32.const 11
    i32.add
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 6800
    i32.store offset=4
    local.get $0
    i32.const 6800
    local.get $1
    call $~lib/string/String#indexOf
    local.tee $4
    local.get $1
    i32.gt_s
    local.get $1
    i32.const 10
    i32.gt_s
    i32.and
    if
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $0
     local.get $1
     local.get $4
     call $~lib/string/String#substring
     local.tee $3
     i32.store offset=16
    end
   end
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=4
   local.get $2
   i32.const 1056
   call $~lib/string/String.__eq
   if
    global.get $~lib/memory/__stack_pointer
    i32.const 20
    i32.add
    global.set $~lib/memory/__stack_pointer
    i32.const 12240
    return
   end
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $3
   i32.store offset=4
   local.get $2
   local.get $3
   f64.const 1024
   f64.const 0.7
   call $assembly/index/performExternalInference
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 20
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $assembly/index/wasmInit
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 13696
  i32.store
  i32.const 13696
  call $~lib/console/console.log
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
 )
 (func $~lib/arraybuffer/ArrayBuffer#constructor (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  local.get $0
  i32.const 1073741820
  i32.gt_u
  if
   i32.const 1488
   i32.const 1536
   i32.const 52
   i32.const 43
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__new
  local.tee $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $~lib/rt/__newArray (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.set $4
  local.get $0
  i32.const 2
  i32.shl
  local.tee $3
  i32.const 1
  call $~lib/rt/itcms/__new
  local.set $2
  local.get $1
  if
   local.get $2
   local.get $1
   local.get $3
   memory.copy
  end
  local.get $4
  local.get $2
  i32.store
  i32.const 16
  i32.const 6
  call $~lib/rt/itcms/__new
  local.tee $1
  local.get $2
  i32.store
  local.get $1
  local.get $2
  i32.const 0
  call $~lib/rt/itcms/__link
  local.get $1
  local.get $2
  i32.store offset=4
  local.get $1
  local.get $3
  i32.store offset=8
  local.get $1
  local.get $0
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $1
 )
 (func $export:assembly/index/createAgentCore (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   local.get $0
   global.set $assembly/index/agentId
   i32.const 0
   global.set $assembly/index/agentInitialized
   global.get $~lib/memory/__stack_pointer
   i32.const 1600
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 1600
   local.get $0
   call $~lib/string/String#concat
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/executeAgent (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 32
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 32
   memory.fill
   block $__inlined_func$assembly/index/executeAgent$298
    global.get $assembly/index/agentInitialized
    i32.eqz
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 32
     i32.add
     global.set $~lib/memory/__stack_pointer
     i32.const 1760
     local.set $0
     br $__inlined_func$assembly/index/executeAgent$298
    end
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=8
    global.get $~lib/memory/__stack_pointer
    i32.const 1984
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=16
    i32.const 1988
    local.get $0
    i32.store
    i32.const 1984
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 1984
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=16
    i32.const 1996
    local.get $1
    i32.store
    i32.const 1984
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 1984
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=16
    i32.const 1984
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $2
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store
    local.get $2
    call $~lib/console/console.log
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/agentId
    local.tee $2
    i32.store offset=24
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=28
    global.get $~lib/memory/__stack_pointer
    i32.const 2304
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=12
    i32.const 2308
    local.get $0
    i32.store
    i32.const 2304
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2304
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=12
    i32.const 2316
    local.get $2
    i32.store
    i32.const 2304
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2304
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=12
    i32.const 2324
    local.get $1
    i32.store
    i32.const 2304
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2304
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=12
    i32.const 2304
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $0
    global.get $~lib/memory/__stack_pointer
    i32.const 32
    i32.add
    global.set $~lib/memory/__stack_pointer
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/executeAgentTool (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 36
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 36
   memory.fill
   block $__inlined_func$assembly/index/executeAgentTool
    global.get $assembly/index/agentInitialized
    i32.eqz
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 36
     i32.add
     global.set $~lib/memory/__stack_pointer
     i32.const 1760
     local.set $0
     br $__inlined_func$assembly/index/executeAgentTool
    end
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=8
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    i32.const 2480
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=20
    i32.const 2484
    local.get $0
    i32.store
    i32.const 2480
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2480
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=20
    i32.const 2492
    local.get $1
    i32.store
    i32.const 2480
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2480
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=20
    i32.const 2500
    local.get $2
    i32.store
    i32.const 2480
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2480
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=20
    i32.const 2480
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $3
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store
    local.get $3
    call $~lib/console/console.log
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=24
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=28
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=32
    global.get $~lib/memory/__stack_pointer
    i32.const 2752
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=16
    i32.const 2756
    local.get $0
    i32.store
    i32.const 2752
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2752
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=16
    i32.const 2764
    local.get $1
    i32.store
    i32.const 2752
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2752
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=16
    i32.const 2772
    local.get $2
    i32.store
    i32.const 2752
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2752
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=16
    i32.const 2752
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $0
    global.get $~lib/memory/__stack_pointer
    i32.const 36
    i32.add
    global.set $~lib/memory/__stack_pointer
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/loadLoraAdapter (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 2800
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 2800
   local.get $0
   call $~lib/string/String#concat
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/createModel (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   local.get $0
   global.set $assembly/index/modelType
   i32.const 0
   global.set $assembly/index/modelLoaded
   global.get $~lib/memory/__stack_pointer
   i32.const 3152
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 3152
   local.get $0
   call $~lib/string/String#concat
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/runModelInference (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 36
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 36
   memory.fill
   block $__inlined_func$assembly/index/runModelInference$1
    global.get $assembly/index/modelLoaded
    i32.eqz
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 36
     i32.add
     global.set $~lib/memory/__stack_pointer
     i32.const 5024
     local.set $0
     br $__inlined_func$assembly/index/runModelInference$1
    end
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/modelType
    local.tee $2
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=8
    global.get $~lib/memory/__stack_pointer
    i32.const 5184
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=16
    i32.const 5188
    local.get $2
    i32.store
    i32.const 5184
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5184
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=16
    i32.const 5196
    local.get $1
    i32.store
    i32.const 5184
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5184
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=16
    i32.const 5184
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $2
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store
    local.get $2
    call $~lib/console/console.log
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/modelType
    local.tee $2
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=24
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/modelType
    local.tee $3
    i32.store offset=28
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=32
    global.get $~lib/memory/__stack_pointer
    i32.const 5472
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=12
    i32.const 5476
    local.get $2
    i32.store
    i32.const 5472
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5472
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=12
    i32.const 5484
    local.get $0
    i32.store
    i32.const 5472
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5472
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $3
    i32.store offset=12
    i32.const 5492
    local.get $3
    i32.store
    i32.const 5472
    local.get $3
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5472
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=12
    i32.const 5500
    local.get $1
    i32.store
    i32.const 5472
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5472
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=12
    i32.const 5472
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $0
    global.get $~lib/memory/__stack_pointer
    i32.const 36
    i32.add
    global.set $~lib/memory/__stack_pointer
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/configureExternalInference (param $0 i32) (param $1 i32) (param $2 i32) (param $3 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=12
  local.get $0
  local.get $1
  local.get $2
  local.get $3
  call $assembly/index/configureExternalInference
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.add
  global.set $~lib/memory/__stack_pointer
  i32.const 1
 )
 (func $export:assembly/index/setActiveInferenceProvider (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  call $assembly/index/setActiveInferenceProvider
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $export:assembly/index/performExternalInference@varargs (param $0 i32) (param $1 i32) (param $2 f64) (param $3 f64) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   block $3of3
    block $2of3
     block $1of3
      block $0of3
       block $outOfRange
        global.get $~argumentsLength
        i32.const 1
        i32.sub
        br_table $0of3 $1of3 $2of3 $3of3 $outOfRange
       end
       unreachable
      end
      i32.const 1056
      local.set $1
      global.get $~lib/memory/__stack_pointer
      i32.const 1056
      i32.store
     end
     f64.const 1024
     local.set $2
    end
    f64.const 0.7
    local.set $3
   end
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=8
   local.get $0
   local.get $1
   local.get $2
   local.get $3
   call $assembly/index/performExternalInference
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/performChatCompletion@varargs (param $0 i32) (param $1 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   block $1of1
    block $0of1
     block $outOfRange
      global.get $~argumentsLength
      i32.const 1
      i32.sub
      br_table $0of1 $1of1 $outOfRange
     end
     unreachable
    end
    i32.const 11904
    local.set $1
    global.get $~lib/memory/__stack_pointer
    i32.const 11904
    i32.store
   end
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=8
   local.get $0
   local.get $1
   call $assembly/index/performChatCompletion
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/initializeExternalInferenceFromEnv (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 13992
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 12400
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 12400
   local.get $0
   call $~lib/string/String#concat
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 6432
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12560
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 12624
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 12752
   i32.store offset=12
   i32.const 6432
   i32.const 12560
   i32.const 12624
   i32.const 12752
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 6512
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12800
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 12864
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 12944
   i32.store offset=12
   i32.const 6512
   i32.const 12800
   i32.const 12864
   i32.const 12944
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 6560
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 12992
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 13056
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 13136
   i32.store offset=12
   i32.const 6560
   i32.const 12992
   i32.const 13056
   i32.const 13136
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 6608
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 13184
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 13248
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 13328
   i32.store offset=12
   i32.const 6608
   i32.const 13184
   i32.const 13248
   i32.const 13328
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 6640
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 13392
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 13456
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 13536
   i32.store offset=12
   i32.const 6640
   i32.const 13392
   i32.const 13456
   i32.const 13536
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 6512
   i32.store
   i32.const 6512
   call $assembly/index/setActiveInferenceProvider
   drop
   global.get $~lib/memory/__stack_pointer
   i32.const 13568
   i32.store
   i32.const 13568
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 1
   return
  end
  i32.const 46784
  i32.const 46832
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/allocateString (param $0 i32) (result f64)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 13992
  i32.lt_s
  if
   i32.const 46784
   i32.const 46832
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
  f64.const 0
 )
)
