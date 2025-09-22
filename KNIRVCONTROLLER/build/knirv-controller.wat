(module
 (type $0 (func (param i32) (result i32)))
 (type $1 (func (param i32 i32) (result i32)))
 (type $2 (func (result i32)))
 (type $3 (func (param i32)))
 (type $4 (func (param i32 i32 i32) (result i32)))
 (type $5 (func))
 (type $6 (func (param i32 i32 i32)))
 (type $7 (func (param i32 i32 i32 i32)))
 (type $8 (func (param i32 i32)))
 (type $9 (func (param i32 i32 i32 i32) (result i32)))
 (type $10 (func (param i32 i32 i64)))
 (type $11 (func (param i32 i32 i32 f32) (result i32)))
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
 (global $~argumentsLength (mut i32) (i32.const 0))
 (global $~lib/rt/__rtti_base i32 (i32.const 12352))
 (global $~lib/memory/__stack_pointer (mut i32) (i32.const 45160))
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
 (data $15 (i32.const 1916) "l")
 (data $15.1 (i32.const 1928) "\02\00\00\00P\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00 \00\"\00r\00e\00s\00u\00l\00t\00\"\00:\00 \00\"\00P\00r\00o\00c\00e\00s\00s\00e\00d\00:\00 ")
 (data $16 (i32.const 2028) "<")
 (data $16.1 (i32.const 2040) "\02\00\00\00\1e\00\00\00\"\00,\00 \00\"\00a\00g\00e\00n\00t\00I\00d\00\"\00:\00 \00\"")
 (data $17 (i32.const 2092) "\1c")
 (data $17.1 (i32.const 2104) "\02\00\00\00\04\00\00\00\"\00}")
 (data $18 (i32.const 2124) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\14\00\00\00\90\07\00\00\00\00\00\00\00\08\00\00\00\00\00\00@\08")
 (data $19 (i32.const 2172) "<")
 (data $19.1 (i32.const 2184) "\02\00\00\00 \00\00\00E\00x\00e\00c\00u\00t\00i\00n\00g\00 \00t\00o\00o\00l\00:\00 ")
 (data $20 (i32.const 2236) "<")
 (data $20.1 (i32.const 2248) "\02\00\00\00$\00\00\00 \00w\00i\00t\00h\00 \00p\00a\00r\00a\00m\00e\00t\00e\00r\00s\00:\00 ")
 (data $21 (i32.const 2300) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\10\00\00\00\90\08\00\00\00\00\00\00\d0\08")
 (data $22 (i32.const 2348) "\\")
 (data $22.1 (i32.const 2360) "\02\00\00\00D\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00 \00\"\00r\00e\00s\00u\00l\00t\00\"\00:\00 \00\"\00T\00o\00o\00l\00 ")
 (data $23 (i32.const 2444) "L")
 (data $23.1 (i32.const 2456) "\02\00\00\004\00\00\00 \00e\00x\00e\00c\00u\00t\00e\00d\00\"\00,\00 \00\"\00p\00a\00r\00a\00m\00e\00t\00e\00r\00s\00\"\00:\00 ")
 (data $24 (i32.const 2524) "\1c")
 (data $24.1 (i32.const 2536) "\02\00\00\00\02\00\00\00}")
 (data $25 (i32.const 2556) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\14\00\00\00@\t\00\00\00\00\00\00\a0\t\00\00\00\00\00\00\f0\t")
 (data $26 (i32.const 2604) "<")
 (data $26.1 (i32.const 2616) "\02\00\00\00,\00\00\00L\00o\00a\00d\00i\00n\00g\00 \00L\00o\00R\00A\00 \00a\00d\00a\00p\00t\00e\00r\00:\00 ")
 (data $27 (i32.const 2668) ",")
 (data $27.1 (i32.const 2680) "\02\00\00\00\1a\00\00\00{\00\"\00a\00g\00e\00n\00t\00I\00d\00\"\00:\00 \00\"")
 (data $28 (i32.const 2716) "<")
 (data $28.1 (i32.const 2728) "\02\00\00\00$\00\00\00\"\00,\00 \00\"\00i\00n\00i\00t\00i\00a\00l\00i\00z\00e\00d\00\"\00:\00 ")
 (data $29 (i32.const 2780) "<")
 (data $29.1 (i32.const 2792) "\02\00\00\00*\00\00\00,\00 \00\"\00v\00e\00r\00s\00i\00o\00n\00\"\00:\00 \00\"\001\00.\000\00.\000\00\"\00}")
 (data $30 (i32.const 2844) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\14\00\00\00\80\n\00\00\00\00\00\00\b0\n\00\00\00\00\00\00\f0\n")
 (data $31 (i32.const 2892) "\1c")
 (data $31.1 (i32.const 2904) "\02\00\00\00\08\00\00\00t\00r\00u\00e")
 (data $32 (i32.const 2924) "\1c")
 (data $32.1 (i32.const 2936) "\02\00\00\00\n\00\00\00f\00a\00l\00s\00e")
 (data $33 (i32.const 2956) "L")
 (data $33.1 (i32.const 2968) "\02\00\00\000\00\00\00C\00r\00e\00a\00t\00i\00n\00g\00 \00n\00e\00w\00 \00M\00o\00d\00e\00l\00W\00A\00S\00M\00:\00 ")
 (data $34 (i32.const 3036) "L")
 (data $34.1 (i32.const 3048) "\02\00\00\006\00\00\00L\00o\00a\00d\00i\00n\00g\00 \00w\00e\00i\00g\00h\00t\00s\00 \00f\00o\00r\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $35 (i32.const 3116) "\1c")
 (data $35.1 (i32.const 3128) "\02\00\00\00\04\00\00\00 \00(")
 (data $36 (i32.const 3148) ",")
 (data $36.1 (i32.const 3160) "\02\00\00\00\0e\00\00\00 \00b\00y\00t\00e\00s\00)")
 (data $37 (i32.const 3196) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\14\00\00\00\f0\0b\00\00\00\00\00\00@\0c\00\00\00\00\00\00`\0c")
 (data $38 (i32.const 3244) "|")
 (data $38.1 (i32.const 3256) "\02\00\00\00d\00\00\00t\00o\00S\00t\00r\00i\00n\00g\00(\00)\00 \00r\00a\00d\00i\00x\00 \00a\00r\00g\00u\00m\00e\00n\00t\00 \00m\00u\00s\00t\00 \00b\00e\00 \00b\00e\00t\00w\00e\00e\00n\00 \002\00 \00a\00n\00d\00 \003\006")
 (data $39 (i32.const 3372) "<")
 (data $39.1 (i32.const 3384) "\02\00\00\00&\00\00\00~\00l\00i\00b\00/\00u\00t\00i\00l\00/\00n\00u\00m\00b\00e\00r\00.\00t\00s")
 (data $40 (i32.const 3436) "\1c")
 (data $40.1 (i32.const 3448) "\02\00\00\00\02\00\00\000")
 (data $41 (i32.const 3468) "0\000\000\001\000\002\000\003\000\004\000\005\000\006\000\007\000\008\000\009\001\000\001\001\001\002\001\003\001\004\001\005\001\006\001\007\001\008\001\009\002\000\002\001\002\002\002\003\002\004\002\005\002\006\002\007\002\008\002\009\003\000\003\001\003\002\003\003\003\004\003\005\003\006\003\007\003\008\003\009\004\000\004\001\004\002\004\003\004\004\004\005\004\006\004\007\004\008\004\009\005\000\005\001\005\002\005\003\005\004\005\005\005\006\005\007\005\008\005\009\006\000\006\001\006\002\006\003\006\004\006\005\006\006\006\007\006\008\006\009\007\000\007\001\007\002\007\003\007\004\007\005\007\006\007\007\007\008\007\009\008\000\008\001\008\002\008\003\008\004\008\005\008\006\008\007\008\008\008\009\009\000\009\001\009\002\009\003\009\004\009\005\009\006\009\007\009\008\009\009")
 (data $42 (i32.const 3868) "\1c\04")
 (data $42.1 (i32.const 3880) "\02\00\00\00\00\04\00\000\000\000\001\000\002\000\003\000\004\000\005\000\006\000\007\000\008\000\009\000\00a\000\00b\000\00c\000\00d\000\00e\000\00f\001\000\001\001\001\002\001\003\001\004\001\005\001\006\001\007\001\008\001\009\001\00a\001\00b\001\00c\001\00d\001\00e\001\00f\002\000\002\001\002\002\002\003\002\004\002\005\002\006\002\007\002\008\002\009\002\00a\002\00b\002\00c\002\00d\002\00e\002\00f\003\000\003\001\003\002\003\003\003\004\003\005\003\006\003\007\003\008\003\009\003\00a\003\00b\003\00c\003\00d\003\00e\003\00f\004\000\004\001\004\002\004\003\004\004\004\005\004\006\004\007\004\008\004\009\004\00a\004\00b\004\00c\004\00d\004\00e\004\00f\005\000\005\001\005\002\005\003\005\004\005\005\005\006\005\007\005\008\005\009\005\00a\005\00b\005\00c\005\00d\005\00e\005\00f\006\000\006\001\006\002\006\003\006\004\006\005\006\006\006\007\006\008\006\009\006\00a\006\00b\006\00c\006\00d\006\00e\006\00f\007\000\007\001\007\002\007\003\007\004\007\005\007\006\007\007\007\008\007\009\007\00a\007\00b\007\00c\007\00d\007\00e\007\00f\008\000\008\001\008\002\008\003\008\004\008\005\008\006\008\007\008\008\008\009\008\00a\008\00b\008\00c\008\00d\008\00e\008\00f\009\000\009\001\009\002\009\003\009\004\009\005\009\006\009\007\009\008\009\009\009\00a\009\00b\009\00c\009\00d\009\00e\009\00f\00a\000\00a\001\00a\002\00a\003\00a\004\00a\005\00a\006\00a\007\00a\008\00a\009\00a\00a\00a\00b\00a\00c\00a\00d\00a\00e\00a\00f\00b\000\00b\001\00b\002\00b\003\00b\004\00b\005\00b\006\00b\007\00b\008\00b\009\00b\00a\00b\00b\00b\00c\00b\00d\00b\00e\00b\00f\00c\000\00c\001\00c\002\00c\003\00c\004\00c\005\00c\006\00c\007\00c\008\00c\009\00c\00a\00c\00b\00c\00c\00c\00d\00c\00e\00c\00f\00d\000\00d\001\00d\002\00d\003\00d\004\00d\005\00d\006\00d\007\00d\008\00d\009\00d\00a\00d\00b\00d\00c\00d\00d\00d\00e\00d\00f\00e\000\00e\001\00e\002\00e\003\00e\004\00e\005\00e\006\00e\007\00e\008\00e\009\00e\00a\00e\00b\00e\00c\00e\00d\00e\00e\00e\00f\00f\000\00f\001\00f\002\00f\003\00f\004\00f\005\00f\006\00f\007\00f\008\00f\009\00f\00a\00f\00b\00f\00c\00f\00d\00f\00e\00f\00f")
 (data $43 (i32.const 4924) "\\")
 (data $43.1 (i32.const 4936) "\02\00\00\00H\00\00\000\001\002\003\004\005\006\007\008\009\00a\00b\00c\00d\00e\00f\00g\00h\00i\00j\00k\00l\00m\00n\00o\00p\00q\00r\00s\00t\00u\00v\00w\00x\00y\00z")
 (data $44 (i32.const 5020) "L")
 (data $44.1 (i32.const 5032) "\02\00\00\00:\00\00\00{\00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00M\00o\00d\00e\00l\00 \00n\00o\00t\00 \00l\00o\00a\00d\00e\00d\00\"\00}")
 (data $45 (i32.const 5100) "L")
 (data $45.1 (i32.const 5112) "\02\00\00\008\00\00\00R\00u\00n\00n\00i\00n\00g\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00o\00n\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $46 (i32.const 5180) "\\")
 (data $46.1 (i32.const 5192) "\02\00\00\00F\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00 \00\"\00r\00e\00s\00u\00l\00t\00\"\00:\00 \00\"\00M\00o\00d\00e\00l\00 ")
 (data $47 (i32.const 5276) "L")
 (data $47.1 (i32.const 5288) "\02\00\00\00.\00\00\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00r\00e\00s\00u\00l\00t\00 \00f\00o\00r\00:\00 ")
 (data $48 (i32.const 5356) "<")
 (data $48.1 (i32.const 5368) "\02\00\00\00\"\00\00\00\"\00,\00 \00\"\00m\00o\00d\00e\00l\00T\00y\00p\00e\00\"\00:\00 \00\"")
 (data $49 (i32.const 5420) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\1c\00\00\00P\14\00\00\00\00\00\00\b0\14\00\00\00\00\00\00\00\15\00\00\00\00\00\00@\08")
 (data $50 (i32.const 5468) "<")
 (data $50.1 (i32.const 5480) "\02\00\00\00\1e\00\00\00{\00\"\00m\00o\00d\00e\00l\00T\00y\00p\00e\00\"\00:\00 \00\"")
 (data $51 (i32.const 5532) ",")
 (data $51.1 (i32.const 5544) "\02\00\00\00\1a\00\00\00\"\00,\00 \00\"\00l\00o\00a\00d\00e\00d\00\"\00:\00 ")
 (data $52 (i32.const 5580) "\ac")
 (data $52.1 (i32.const 5592) "\02\00\00\00\92\00\00\00,\00 \00\"\00c\00a\00p\00a\00b\00i\00l\00i\00t\00i\00e\00s\00\"\00:\00 \00[\00\"\00t\00e\00x\00t\00-\00g\00e\00n\00e\00r\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00,\00 \00\"\00e\00x\00t\00e\00r\00n\00a\00l\00-\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00]\00}")
 (data $53 (i32.const 5756) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\14\00\00\00p\15\00\00\00\00\00\00\b0\15\00\00\00\00\00\00\e0\15")
 (data $54 (i32.const 5804) "l")
 (data $54.1 (i32.const 5816) "\02\00\00\00Z\00\00\00C\00o\00n\00f\00i\00g\00u\00r\00i\00n\00g\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00f\00o\00r\00 \00p\00r\00o\00v\00i\00d\00e\00r\00:\00 ")
 (data $55 (i32.const 5916) ",")
 (data $55.1 (i32.const 5928) "\02\00\00\00\12\00\00\00P\00r\00o\00v\00i\00d\00e\00r\00 ")
 (data $56 (i32.const 5964) "L")
 (data $56.1 (i32.const 5976) "\02\00\00\006\00\00\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00 \00w\00i\00t\00h\00 \00e\00n\00d\00p\00o\00i\00n\00t\00:\00 ")
 (data $57 (i32.const 6044) ",")
 (data $57.1 (i32.const 6056) "\02\00\00\00\12\00\00\00,\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $58 (i32.const 6092) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\18\00\00\000\17\00\00\00\00\00\00`\17\00\00\00\00\00\00\b0\17")
 (data $59 (i32.const 6140) "\\")
 (data $59.1 (i32.const 6152) "\02\00\00\00D\00\00\00A\00c\00t\00i\00v\00e\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00p\00r\00o\00v\00i\00d\00e\00r\00 \00s\00e\00t\00 \00t\00o\00:\00 ")
 (data $60 (i32.const 6236) "<")
 (data $60.1 (i32.const 6248) "\02\00\00\00\1e\00\00\00 \00n\00o\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d")
 (data $61 (i32.const 6300) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\000\17\00\00\00\00\00\00p\18")
 (data $62 (i32.const 6332) "\1c")
 (data $62.1 (i32.const 6344) "\01")
 (data $63 (i32.const 6364) ",")
 (data $63.1 (i32.const 6376) "\02\00\00\00\1a\00\00\00~\00l\00i\00b\00/\00a\00r\00r\00a\00y\00.\00t\00s")
 (data $64 (i32.const 6412) "|")
 (data $64.1 (i32.const 6424) "\02\00\00\00^\00\00\00E\00l\00e\00m\00e\00n\00t\00 \00t\00y\00p\00e\00 \00m\00u\00s\00t\00 \00b\00e\00 \00n\00u\00l\00l\00a\00b\00l\00e\00 \00i\00f\00 \00a\00r\00r\00a\00y\00 \00i\00s\00 \00h\00o\00l\00e\00y")
 (data $65 (i32.const 6540) "<")
 (data $65.1 (i32.const 6552) "\02\00\00\00\1e\00\00\00{\00\"\00p\00r\00o\00v\00i\00d\00e\00r\00s\00\"\00:\00 \00[")
 (data $66 (i32.const 6604) "\1c")
 (data $66.1 (i32.const 6616) "\02\00\00\00\04\00\00\00]\00}")
 (data $67 (i32.const 6636) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\a0\19\00\00\00\00\00\00\e0\19")
 (data $68 (i32.const 6668) "\1c")
 (data $68.1 (i32.const 6680) "\02\00\00\00\02\00\00\00\"")
 (data $69 (i32.const 6700) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00 \1a\00\00\00\00\00\00 \1a")
 (data $70 (i32.const 6732) "\1c")
 (data $70.1 (i32.const 6744) "\08\00\00\00\08\00\00\00\01")
 (data $71 (i32.const 6764) "\1c")
 (data $71.1 (i32.const 6776) "\02\00\00\00\04\00\00\00,\00 ")
 (data $72 (i32.const 6796) "\9c")
 (data $72.1 (i32.const 6808) "\02\00\00\00\80\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00f\00a\00l\00s\00e\00,\00 \00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00E\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00n\00o\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00\"\00}")
 (data $73 (i32.const 6956) "\8c")
 (data $73.1 (i32.const 6968) "\02\00\00\00z\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00f\00a\00l\00s\00e\00,\00 \00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00A\00c\00t\00i\00v\00e\00 \00p\00r\00o\00v\00i\00d\00e\00r\00 \00n\00o\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00\"\00}")
 (data $74 (i32.const 7100) "<")
 (data $74.1 (i32.const 7112) "\02\00\00\00$\00\00\00K\00e\00y\00 \00d\00o\00e\00s\00 \00n\00o\00t\00 \00e\00x\00i\00s\00t")
 (data $75 (i32.const 7164) ",")
 (data $75.1 (i32.const 7176) "\02\00\00\00\16\00\00\00~\00l\00i\00b\00/\00m\00a\00p\00.\00t\00s")
 (data $76 (i32.const 7212) "\\")
 (data $76.1 (i32.const 7224) "\02\00\00\00F\00\00\00P\00e\00r\00f\00o\00r\00m\00i\00n\00g\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00w\00i\00t\00h\00 ")
 (data $77 (i32.const 7308) ",")
 (data $77.1 (i32.const 7320) "\02\00\00\00\1c\00\00\00 \00u\00s\00i\00n\00g\00 \00m\00o\00d\00e\00l\00:\00 ")
 (data $78 (i32.const 7356) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\10\00\00\00@\1c\00\00\00\00\00\00\a0\1c")
 (data $79 (i32.const 7404) "\1c")
 (data $79.1 (i32.const 7416) "\02\00\00\00\0c\00\00\00g\00e\00m\00i\00n\00i")
 (data $80 (i32.const 7436) "L")
 (data $80.1 (i32.const 7448) "\02\00\00\000\00\00\00G\00e\00m\00i\00n\00i\00 \00A\00I\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $81 (i32.const 7516) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00 \1d\00\00\00\00\00\00 \1a")
 (data $82 (i32.const 7548) "L")
 (data $82.1 (i32.const 7560) "\02\00\00\008\00\00\00 \00(\00f\00o\00l\00l\00o\00w\00i\00n\00g\00 \00s\00y\00s\00t\00e\00m\00 \00p\00r\00o\00m\00p\00t\00:\00 \00\"")
 (data $83 (i32.const 7628) "\1c")
 (data $83.1 (i32.const 7640) "\02\00\00\00\04\00\00\00\"\00)")
 (data $84 (i32.const 7660) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\90\1d\00\00\00\00\00\00\e0\1d")
 (data $85 (i32.const 7692) ",")
 (data $85.1 (i32.const 7704) "\02\00\00\00\10\00\00\00c\00e\00r\00e\00b\00r\00a\00s")
 (data $86 (i32.const 7740) "|")
 (data $86.1 (i32.const 7752) "\02\00\00\00^\00\00\00C\00e\00r\00e\00b\00r\00a\00s\00 \00h\00i\00g\00h\00-\00p\00e\00r\00f\00o\00r\00m\00a\00n\00c\00e\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00r\00e\00s\00p\00o\00n\00s\00e\00:\00 \00\"")
 (data $87 (i32.const 7868) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00P\1e\00\00\00\00\00\00 \1a")
 (data $88 (i32.const 7900) ",")
 (data $88.1 (i32.const 7912) "\02\00\00\00\16\00\00\00 \00(\00s\00y\00s\00t\00e\00m\00:\00 \00\"")
 (data $89 (i32.const 7948) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\f0\1e\00\00\00\00\00\00\e0\1d")
 (data $90 (i32.const 7980) ",")
 (data $90.1 (i32.const 7992) "\02\00\00\00\10\00\00\00d\00e\00e\00p\00s\00e\00e\00k")
 (data $91 (i32.const 8028) "\\")
 (data $91.1 (i32.const 8040) "\02\00\00\00B\00\00\00D\00e\00e\00p\00S\00e\00e\00k\00 \00r\00e\00a\00s\00o\00n\00i\00n\00g\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $92 (i32.const 8124) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00p\1f\00\00\00\00\00\00 \1a")
 (data $93 (i32.const 8156) ",")
 (data $93.1 (i32.const 8168) "\02\00\00\00\1c\00\00\00 \00(\00g\00u\00i\00d\00e\00d\00 \00b\00y\00:\00 \00\"")
 (data $94 (i32.const 8204) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\f0\1f\00\00\00\00\00\00\e0\1d")
 (data $95 (i32.const 8236) "\1c")
 (data $95.1 (i32.const 8248) "\02\00\00\00\0c\00\00\00c\00l\00a\00u\00d\00e")
 (data $96 (i32.const 8268) "\\")
 (data $96.1 (i32.const 8280) "\02\00\00\00H\00\00\00C\00l\00a\00u\00d\00e\00 \00c\00o\00n\00s\00t\00i\00t\00u\00t\00i\00o\00n\00a\00l\00 \00A\00I\00 \00r\00e\00s\00p\00o\00n\00s\00e\00:\00 \00\"")
 (data $97 (i32.const 8364) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00` \00\00\00\00\00\00 \1a")
 (data $98 (i32.const 8396) "L")
 (data $98.1 (i32.const 8408) "\02\00\00\002\00\00\00 \00(\00w\00i\00t\00h\00 \00s\00y\00s\00t\00e\00m\00 \00g\00u\00i\00d\00a\00n\00c\00e\00:\00 \00\"")
 (data $99 (i32.const 8476) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\e0 \00\00\00\00\00\00\e0\1d")
 (data $100 (i32.const 8508) "\1c")
 (data $100.1 (i32.const 8520) "\02\00\00\00\0c\00\00\00o\00p\00e\00n\00a\00i")
 (data $101 (i32.const 8540) "L")
 (data $101.1 (i32.const 8552) "\02\00\00\002\00\00\00O\00p\00e\00n\00A\00I\00 \00G\00P\00T\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $102 (i32.const 8620) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00p!\00\00\00\00\00\00 \1a")
 (data $103 (i32.const 8652) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\f0\1e\00\00\00\00\00\00\e0\1d")
 (data $104 (i32.const 8684) "\\")
 (data $104.1 (i32.const 8696) "\02\00\00\00>\00\00\00U\00n\00k\00n\00o\00w\00n\00 \00p\00r\00o\00v\00i\00d\00e\00r\00 \00r\00e\00s\00p\00o\00n\00s\00e\00 \00t\00o\00:\00 \00\"")
 (data $105 (i32.const 8780) "\1c\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\0c\00\00\00\00\"\00\00\00\00\00\00 \1a")
 (data $106 (i32.const 8812) "l")
 (data $106.1 (i32.const 8824) "\02\00\00\00N\00\00\00{\00\n\00 \00 \00 \00 \00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00t\00r\00u\00e\00,\00\n\00 \00 \00 \00 \00\"\00c\00o\00n\00t\00e\00n\00t\00\"\00:\00 \00\"")
 (data $107 (i32.const 8924) "<")
 (data $107.1 (i32.const 8936) "\02\00\00\00(\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00p\00r\00o\00v\00i\00d\00e\00r\00\"\00:\00 \00\"")
 (data $108 (i32.const 8988) "<")
 (data $108.1 (i32.const 9000) "\02\00\00\00\"\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00m\00o\00d\00e\00l\00\"\00:\00 \00\"")
 (data $109 (i32.const 9052) "\9c")
 (data $109.1 (i32.const 9064) "\02\00\00\00\8a\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00p\00r\00o\00c\00e\00s\00s\00i\00n\00g\00T\00i\00m\00e\00\"\00:\00 \001\005\000\00.\005\00,\00\n\00 \00 \00 \00 \00\"\00u\00s\00a\00g\00e\00\"\00:\00 \00{\00\n\00 \00 \00 \00 \00 \00 \00\"\00p\00r\00o\00m\00p\00t\00T\00o\00k\00e\00n\00s\00\"\00:\00 ")
 (data $110 (i32.const 9212) "L")
 (data $110.1 (i32.const 9224) "\02\00\00\008\00\00\00,\00\n\00 \00 \00 \00 \00 \00 \00\"\00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00T\00o\00k\00e\00n\00s\00\"\00:\00 ")
 (data $111 (i32.const 9292) "L")
 (data $111.1 (i32.const 9304) "\02\00\00\00.\00\00\00,\00\n\00 \00 \00 \00 \00 \00 \00\"\00t\00o\00t\00a\00l\00T\00o\00k\00e\00n\00s\00\"\00:\00 ")
 (data $112 (i32.const 9372) ",")
 (data $112.1 (i32.const 9384) "\02\00\00\00\14\00\00\00\n\00 \00 \00 \00 \00}\00\n\00 \00 \00}")
 (data $113 (i32.const 9420) "L\00\00\00\03\00\00\00\00\00\00\00\05\00\00\004\00\00\00\80\"\00\00\00\00\00\00\f0\"\00\00\00\00\00\000#\00\00\00\00\00\00p#\00\00\00\00\00\00\10$\00\00\00\00\00\00`$\00\00\00\00\00\00\b0$")
 (data $114 (i32.const 9500) "<")
 (data $114.1 (i32.const 9512) "\02\00\00\00\"\00\00\00{\00\n\00 \00 \00 \00 \00\"\00e\00n\00a\00b\00l\00e\00d\00\"\00:\00 ")
 (data $115 (i32.const 9564) "L")
 (data $115.1 (i32.const 9576) "\02\00\00\002\00\00\00,\00\n\00 \00 \00 \00 \00\"\00a\00c\00t\00i\00v\00e\00P\00r\00o\00v\00i\00d\00e\00r\00\"\00:\00 \00\"")
 (data $116 (i32.const 9644) "L")
 (data $116.1 (i32.const 9656) "\02\00\00\00<\00\00\00\"\00,\00\n\00 \00 \00 \00 \00\"\00c\00o\00n\00f\00i\00g\00u\00r\00e\00d\00P\00r\00o\00v\00i\00d\00e\00r\00s\00\"\00:\00 ")
 (data $117 (i32.const 9724) "\bc")
 (data $117.1 (i32.const 9736) "\02\00\00\00\a0\00\00\00,\00\n\00 \00 \00 \00 \00\"\00c\00a\00p\00a\00b\00i\00l\00i\00t\00i\00e\00s\00\"\00:\00 \00[\00\"\00c\00h\00a\00t\00-\00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00\"\00,\00 \00\"\00t\00e\00x\00t\00-\00g\00e\00n\00e\00r\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00c\00o\00n\00v\00e\00r\00s\00a\00t\00i\00o\00n\00\"\00]\00\n\00 \00 \00}")
 (data $118 (i32.const 9916) ",\00\00\00\03\00\00\00\00\00\00\00\05\00\00\00\1c\00\00\000%\00\00\00\00\00\00p%\00\00\00\00\00\00\c0%\00\00\00\00\00\00\10&")
 (data $119 (i32.const 9964) "\1c")
 (data $119.1 (i32.const 9976) "\02\00\00\00\n\00\00\001\00.\000\00.\000")
 (data $120 (i32.const 9996) "<\01")
 (data $120.1 (i32.const 10008) "\02\00\00\00&\01\00\00[\00\"\00a\00g\00e\00n\00t\00-\00c\00o\00r\00e\00\"\00,\00 \00\"\00m\00o\00d\00e\00l\00-\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00,\00 \00\"\00l\00o\00r\00a\00-\00a\00d\00a\00p\00t\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00c\00r\00o\00s\00s\00-\00w\00a\00s\00m\00-\00c\00o\00m\00m\00u\00n\00i\00c\00a\00t\00i\00o\00n\00\"\00,\00 \00\"\00e\00x\00t\00e\00r\00n\00a\00l\00-\00i\00n\00f\00e\00r\00e\00n\00c\00e\00\"\00,\00 \00\"\00c\00h\00a\00t\00-\00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00\"\00,\00 \00\"\00m\00u\00l\00t\00i\00-\00p\00r\00o\00v\00i\00d\00e\00r\00-\00s\00u\00p\00p\00o\00r\00t\00\"\00]")
 (data $121 (i32.const 10316) "\1c")
 (data $121.1 (i32.const 10328) "\02\00\00\00\04\00\00\00{\00}")
 (data $122 (i32.const 10348) "|")
 (data $122.1 (i32.const 10360) "\02\00\00\00d\00\00\00P\00e\00r\00f\00o\00r\00m\00i\00n\00g\00 \00c\00h\00a\00t\00 \00c\00o\00m\00p\00l\00e\00t\00i\00o\00n\00 \00w\00i\00t\00h\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e")
 (data $123 (i32.const 10476) ",")
 (data $123.1 (i32.const 10488) "\02\00\00\00\1a\00\00\00\"\00r\00o\00l\00e\00\"\00:\00\"\00u\00s\00e\00r\00\"")
 (data $124 (i32.const 10524) ",")
 (data $124.1 (i32.const 10536) "\02\00\00\00\16\00\00\00\"\00c\00o\00n\00t\00e\00n\00t\00\"\00:\00\"")
 (data $125 (i32.const 10572) "<")
 (data $125.1 (i32.const 10584) "\02\00\00\00\1e\00\00\00\"\00r\00o\00l\00e\00\"\00:\00\"\00s\00y\00s\00t\00e\00m\00\"")
 (data $126 (i32.const 10636) "\9c")
 (data $126.1 (i32.const 10648) "\02\00\00\00\88\00\00\00{\00\"\00s\00u\00c\00c\00e\00s\00s\00\"\00:\00 \00f\00a\00l\00s\00e\00,\00 \00\"\00e\00r\00r\00o\00r\00\"\00:\00 \00\"\00N\00o\00 \00u\00s\00e\00r\00 \00m\00e\00s\00s\00a\00g\00e\00 \00f\00o\00u\00n\00d\00 \00i\00n\00 \00c\00o\00n\00v\00e\00r\00s\00a\00t\00i\00o\00n\00\"\00}")
 (data $127 (i32.const 10796) "\8c")
 (data $127.1 (i32.const 10808) "\02\00\00\00|\00\00\00I\00n\00i\00t\00i\00a\00l\00i\00z\00i\00n\00g\00 \00e\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00f\00r\00o\00m\00 \00e\00n\00v\00i\00r\00o\00n\00m\00e\00n\00t\00 \00c\00o\00n\00f\00i\00g\00u\00r\00a\00t\00i\00o\00n")
 (data $128 (i32.const 10940) "<")
 (data $128.1 (i32.const 10952) "\02\00\00\00\1e\00\00\00t\00e\00s\00t\00-\00g\00e\00m\00i\00n\00i\00-\00k\00e\00y")
 (data $129 (i32.const 11004) "|")
 (data $129.1 (i32.const 11016) "\02\00\00\00`\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00g\00e\00n\00e\00r\00a\00t\00i\00v\00e\00l\00a\00n\00g\00u\00a\00g\00e\00.\00g\00o\00o\00g\00l\00e\00a\00p\00i\00s\00.\00c\00o\00m\00/\00v\001\00b\00e\00t\00a")
 (data $130 (i32.const 11132) ",")
 (data $130.1 (i32.const 11144) "\02\00\00\00\14\00\00\00g\00e\00m\00i\00n\00i\00-\00p\00r\00o")
 (data $131 (i32.const 11180) "<")
 (data $131.1 (i32.const 11192) "\02\00\00\00\"\00\00\00t\00e\00s\00t\00-\00c\00e\00r\00e\00b\00r\00a\00s\00-\00k\00e\00y")
 (data $132 (i32.const 11244) "L")
 (data $132.1 (i32.const 11256) "\02\00\00\004\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00c\00e\00r\00e\00b\00r\00a\00s\00.\00a\00i\00/\00v\001")
 (data $133 (i32.const 11324) ",")
 (data $133.1 (i32.const 11336) "\02\00\00\00\16\00\00\00l\00l\00a\00m\00a\003\00.\001\00-\008\00b")
 (data $134 (i32.const 11372) "<")
 (data $134.1 (i32.const 11384) "\02\00\00\00\"\00\00\00t\00e\00s\00t\00-\00d\00e\00e\00p\00s\00e\00e\00k\00-\00k\00e\00y")
 (data $135 (i32.const 11436) "L")
 (data $135.1 (i32.const 11448) "\02\00\00\006\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00d\00e\00e\00p\00s\00e\00e\00k\00.\00c\00o\00m\00/\00v\001")
 (data $136 (i32.const 11516) ",")
 (data $136.1 (i32.const 11528) "\02\00\00\00\1a\00\00\00d\00e\00e\00p\00s\00e\00e\00k\00-\00c\00h\00a\00t")
 (data $137 (i32.const 11564) "<")
 (data $137.1 (i32.const 11576) "\02\00\00\00\1e\00\00\00t\00e\00s\00t\00-\00c\00l\00a\00u\00d\00e\00-\00k\00e\00y")
 (data $138 (i32.const 11628) "L")
 (data $138.1 (i32.const 11640) "\02\00\00\008\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00a\00n\00t\00h\00r\00o\00p\00i\00c\00.\00c\00o\00m\00/\00v\001")
 (data $139 (i32.const 11708) "<")
 (data $139.1 (i32.const 11720) "\02\00\00\00\1e\00\00\00c\00l\00a\00u\00d\00e\00-\003\00-\00s\00o\00n\00n\00e\00t")
 (data $140 (i32.const 11772) "<")
 (data $140.1 (i32.const 11784) "\02\00\00\00\1e\00\00\00t\00e\00s\00t\00-\00o\00p\00e\00n\00a\00i\00-\00k\00e\00y")
 (data $141 (i32.const 11836) "L")
 (data $141.1 (i32.const 11848) "\02\00\00\002\00\00\00h\00t\00t\00p\00s\00:\00/\00/\00a\00p\00i\00.\00o\00p\00e\00n\00a\00i\00.\00c\00o\00m\00/\00v\001")
 (data $142 (i32.const 11916) "\1c")
 (data $142.1 (i32.const 11928) "\02\00\00\00\n\00\00\00g\00p\00t\00-\004")
 (data $143 (i32.const 11948) "|")
 (data $143.1 (i32.const 11960) "\02\00\00\00l\00\00\00E\00x\00t\00e\00r\00n\00a\00l\00 \00i\00n\00f\00e\00r\00e\00n\00c\00e\00 \00i\00n\00i\00t\00i\00a\00l\00i\00z\00e\00d\00 \00w\00i\00t\00h\00 \00m\00u\00l\00t\00i\00p\00l\00e\00 \00p\00r\00o\00v\00i\00d\00e\00r\00s")
 (data $144 (i32.const 12076) "\8c")
 (data $144.1 (i32.const 12088) "\02\00\00\00n\00\00\00K\00N\00I\00R\00V\00 \00C\00o\00n\00t\00r\00o\00l\00l\00e\00r\00 \00A\00s\00s\00e\00m\00b\00l\00y\00S\00c\00r\00i\00p\00t\00 \00W\00A\00S\00M\00 \00m\00o\00d\00u\00l\00e\00 \00i\00n\00i\00t\00i\00a\00l\00i\00z\00e\00d")
 (data $145 (i32.const 12220) "<")
 (data $145.1 (i32.const 12232) "\02\00\00\00*\00\00\00O\00b\00j\00e\00c\00t\00 \00a\00l\00r\00e\00a\00d\00y\00 \00p\00i\00n\00n\00e\00d")
 (data $146 (i32.const 12284) "<")
 (data $146.1 (i32.const 12296) "\02\00\00\00(\00\00\00O\00b\00j\00e\00c\00t\00 \00i\00s\00 \00n\00o\00t\00 \00p\00i\00n\00n\00e\00d")
 (data $147 (i32.const 12352) "\t\00\00\00 \00\00\00 \00\00\00 \00\00\00\00\00\00\00\10A\82\00\04A\00\00\02A\00\00\02\t")
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
  i32.const 6432
  call $~lib/rt/itcms/__visit
  i32.const 7120
  call $~lib/rt/itcms/__visit
  i32.const 1088
  call $~lib/rt/itcms/__visit
  i32.const 12240
  call $~lib/rt/itcms/__visit
  i32.const 12304
  call $~lib/rt/itcms/__visit
  i32.const 3888
  call $~lib/rt/itcms/__visit
  i32.const 4944
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
   i32.const 12352
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
   i32.const 12356
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
  i32.const 45168
  i32.const 0
  i32.store
  i32.const 46736
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
    i32.const 45168
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
      i32.const 45168
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
  i32.const 45168
  i32.const 46740
  memory.size
  i64.extend_i32_s
  i64.const 16
  i64.shl
  call $~lib/rt/tlsf/addMemory
  i32.const 45168
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
      i32.const 45160
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
    i32.const 45160
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
     i32.const 45160
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
 (func $~lib/number/I32#toString (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  block $__inlined_func$~lib/util/number/itoa32$75
   local.get $0
   i32.eqz
   if
    global.get $~lib/memory/__stack_pointer
    i32.const 4
    i32.add
    global.set $~lib/memory/__stack_pointer
    i32.const 3456
    local.set $2
    br $__inlined_func$~lib/util/number/itoa32$75
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
   local.tee $4
   select
   local.tee $0
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
   local.tee $1
   i32.const 1
   i32.shl
   local.get $4
   i32.add
   i32.const 2
   call $~lib/rt/itcms/__new
   local.tee $2
   i32.store
   local.get $2
   local.get $4
   i32.add
   local.set $3
   loop $while-continue|0
    local.get $0
    i32.const 10000
    i32.ge_u
    if
     local.get $0
     i32.const 10000
     i32.rem_u
     local.set $5
     local.get $0
     i32.const 10000
     i32.div_u
     local.set $0
     local.get $3
     local.get $1
     i32.const 4
     i32.sub
     local.tee $1
     i32.const 1
     i32.shl
     i32.add
     local.get $5
     i32.const 100
     i32.div_u
     i32.const 2
     i32.shl
     i32.const 3468
     i32.add
     i64.load32_u
     local.get $5
     i32.const 100
     i32.rem_u
     i32.const 2
     i32.shl
     i32.const 3468
     i32.add
     i64.load32_u
     i64.const 32
     i64.shl
     i64.or
     i64.store
     br $while-continue|0
    end
   end
   local.get $0
   i32.const 100
   i32.ge_u
   if
    local.get $3
    local.get $1
    i32.const 2
    i32.sub
    local.tee $1
    i32.const 1
    i32.shl
    i32.add
    local.get $0
    i32.const 100
    i32.rem_u
    i32.const 2
    i32.shl
    i32.const 3468
    i32.add
    i32.load
    i32.store
    local.get $0
    i32.const 100
    i32.div_u
    local.set $0
   end
   local.get $0
   i32.const 10
   i32.ge_u
   if
    local.get $3
    local.get $1
    i32.const 2
    i32.sub
    i32.const 1
    i32.shl
    i32.add
    local.get $0
    i32.const 2
    i32.shl
    i32.const 3468
    i32.add
    i32.load
    i32.store
   else
    local.get $3
    local.get $1
    i32.const 1
    i32.sub
    i32.const 1
    i32.shl
    i32.add
    local.get $0
    i32.const 48
    i32.add
    i32.store16
   end
   local.get $4
   if
    local.get $2
    i32.const 45
    i32.store16
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
  end
  local.get $2
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
 (func $assembly/index/getWasmVersion (result i32)
  i32.const 9984
 )
 (func $assembly/index/getSupportedFeatures (result i32)
  i32.const 10016
 )
 (func $assembly/index/deallocateString (param $0 i32)
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
    i32.const 12240
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
   i32.const 12304
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
          i32.const 12392
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
        i32.const 12392
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
       i32.const 12392
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
      i32.const 12392
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
    i32.const 45184
    i32.const 45232
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
  i32.const 45160
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 2912
  i32.const 2944
  global.get $assembly/index/agentInitialized
  select
  local.tee $1
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 2864
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=12
  i32.const 2868
  local.get $0
  i32.store
  i32.const 2864
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 2864
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=12
  i32.const 2876
  local.get $1
  i32.store
  i32.const 2864
  local.get $1
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 2864
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=12
  i32.const 2864
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.add
  global.set $~lib/memory/__stack_pointer
  local.get $0
 )
 (func $assembly/index/loadModelWeights (param $0 i32) (param $1 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 20
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.const 20
  memory.fill
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/modelType
  local.tee $0
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $1
  call $~lib/number/I32#toString
  local.tee $1
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  i32.const 3216
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=16
  i32.const 3220
  local.get $0
  i32.store
  i32.const 3216
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 3216
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=16
  i32.const 3228
  local.get $1
  i32.store
  i32.const 3216
  local.get $1
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 3216
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=16
  i32.const 3216
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store
  local.get $0
  call $~lib/console/console.log
  i32.const 1
  global.set $assembly/index/modelLoaded
  global.get $~lib/memory/__stack_pointer
  i32.const 20
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 2912
  i32.const 2944
  global.get $assembly/index/modelLoaded
  select
  local.tee $1
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 5776
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=12
  i32.const 5780
  local.get $0
  i32.store
  i32.const 5776
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 5776
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=12
  i32.const 5788
  local.get $1
  i32.store
  i32.const 5776
  local.get $1
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 5776
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=12
  i32.const 5776
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
   i32.const 12392
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
     i32.const 12392
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
  i32.const 45184
  i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 5824
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=8
  i32.const 5824
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
  i32.const 6112
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=8
  i32.const 6116
  local.get $0
  i32.store
  i32.const 6112
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6112
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=8
  i32.const 6124
  local.get $2
  i32.store
  i32.const 6112
  local.get $2
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6112
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=8
  i32.const 6132
  local.get $3
  i32.store
  i32.const 6112
  local.get $3
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6112
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=8
  i32.const 6112
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
   i32.const 6160
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 6160
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
   i32.const 6320
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 6324
   local.get $0
   i32.store
   i32.const 6320
   local.get $0
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 6320
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=8
   i32.const 6320
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
 (func $~lib/array/ensureCapacity (param $0 i32) (param $1 i32) (param $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  local.get $1
  local.get $0
  i32.load offset=8
  local.tee $4
  i32.const 2
  i32.shr_u
  i32.gt_u
  if
   local.get $1
   i32.const 268435455
   i32.gt_u
   if
    i32.const 1488
    i32.const 6384
    i32.const 19
    i32.const 48
    call $~lib/builtins/abort
    unreachable
   end
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   i32.const 8
   local.get $1
   local.get $1
   i32.const 8
   i32.le_u
   select
   i32.const 2
   i32.shl
   local.set $1
   local.get $2
   if
    i32.const 1073741820
    local.get $4
    i32.const 1
    i32.shl
    local.tee $2
    local.get $2
    i32.const 1073741820
    i32.ge_u
    select
    local.tee $2
    local.get $1
    local.get $1
    local.get $2
    i32.lt_u
    select
    local.set $1
   end
   block $__inlined_func$~lib/rt/itcms/__renew$303
    local.get $0
    i32.load
    local.tee $3
    i32.const 20
    i32.sub
    local.tee $5
    i32.load
    i32.const -4
    i32.and
    i32.const 16
    i32.sub
    local.get $1
    i32.ge_u
    if
     local.get $5
     local.get $1
     i32.store offset=16
     local.get $3
     local.set $2
     br $__inlined_func$~lib/rt/itcms/__renew$303
    end
    local.get $1
    local.get $5
    i32.load offset=12
    call $~lib/rt/itcms/__new
    local.tee $2
    local.get $3
    local.get $1
    local.get $5
    i32.load offset=16
    local.tee $4
    local.get $1
    local.get $4
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
   local.get $1
   i32.store offset=8
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.add
  global.set $~lib/memory/__stack_pointer
 )
 (func $"~lib/map/Map<~lib/string/String,~lib/string/String>#keys" (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 i32)
  (local $7 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner1
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner1
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   i32.load offset=8
   local.set $5
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   local.get $0
   i32.load offset=16
   local.set $3
   global.get $~lib/memory/__stack_pointer
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner1
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.const 6
   call $~lib/rt/itcms/__new
   local.tee $2
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   local.get $2
   i32.const 0
   i32.store
   local.get $2
   i32.const 0
   i32.const 0
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   local.get $2
   i32.const 0
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   local.get $2
   i32.const 0
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   local.get $2
   i32.const 0
   i32.store offset=12
   local.get $3
   i32.const 268435455
   i32.gt_u
   if
    i32.const 1488
    i32.const 6384
    i32.const 70
    i32.const 60
    call $~lib/builtins/abort
    unreachable
   end
   global.get $~lib/memory/__stack_pointer
   i32.const 8
   local.get $3
   local.get $3
   i32.const 8
   i32.le_u
   select
   i32.const 2
   i32.shl
   local.tee $6
   i32.const 1
   call $~lib/rt/itcms/__new
   local.tee $7
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $7
   i32.store offset=12
   local.get $2
   local.get $7
   i32.store
   local.get $2
   local.get $7
   i32.const 0
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   local.get $2
   local.get $7
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   local.get $2
   local.get $6
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=4
   local.get $2
   local.get $3
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   local.get $2
   i32.store offset=4
   loop $for-loop|0
    local.get $3
    local.get $4
    i32.gt_s
    if
     local.get $5
     local.get $4
     i32.const 12
     i32.mul
     i32.add
     local.tee $6
     i32.load offset=8
     i32.const 1
     i32.and
     i32.eqz
     if
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store
      local.get $1
      local.tee $0
      i32.const 1
      i32.add
      local.set $1
      global.get $~lib/memory/__stack_pointer
      local.get $6
      i32.load
      local.tee $6
      i32.store offset=8
      global.get $~lib/memory/__stack_pointer
      i32.const 4
      i32.sub
      global.set $~lib/memory/__stack_pointer
      global.get $~lib/memory/__stack_pointer
      i32.const 12392
      i32.lt_s
      br_if $folding-inner1
      global.get $~lib/memory/__stack_pointer
      i32.const 0
      i32.store
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store
      local.get $0
      local.get $2
      i32.load offset=12
      i32.ge_u
      if
       local.get $0
       i32.const 0
       i32.lt_s
       if
        i32.const 1280
        i32.const 6384
        i32.const 130
        i32.const 22
        call $~lib/builtins/abort
        unreachable
       end
       local.get $2
       local.get $0
       i32.const 1
       i32.add
       local.tee $7
       i32.const 1
       call $~lib/array/ensureCapacity
       global.get $~lib/memory/__stack_pointer
       local.get $2
       i32.store
       local.get $2
       local.get $7
       i32.store offset=12
      end
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store
      local.get $2
      i32.load offset=4
      local.get $0
      i32.const 2
      i32.shl
      i32.add
      local.get $6
      i32.store
      local.get $2
      local.get $6
      i32.const 1
      call $~lib/rt/itcms/__link
      global.get $~lib/memory/__stack_pointer
      i32.const 4
      i32.add
      global.set $~lib/memory/__stack_pointer
     end
     local.get $4
     i32.const 1
     i32.add
     local.set $4
     br $for-loop|0
    end
   end
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner1
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store
   local.get $2
   local.get $1
   i32.const 0
   call $~lib/array/ensureCapacity
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store
   local.get $2
   local.get $1
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $2
   return
  end
  i32.const 45184
  i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 6720
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=8
  i32.const 6724
  local.get $0
  i32.store
  i32.const 6720
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 6720
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=8
  i32.const 6720
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
  i32.const 28
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 28
   memory.fill
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 6352
   call $~lib/rt/__newArray
   local.tee $1
   i32.store
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/apiKeys
   local.tee $2
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $2
   call $"~lib/map/Map<~lib/string/String,~lib/string/String>#keys"
   local.tee $2
   i32.store offset=8
   loop $for-loop|0
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 4
    i32.sub
    global.set $~lib/memory/__stack_pointer
    global.get $~lib/memory/__stack_pointer
    i32.const 12392
    i32.lt_s
    br_if $folding-inner0
    global.get $~lib/memory/__stack_pointer
    i32.const 0
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store
    local.get $2
    i32.load offset=12
    local.set $3
    global.get $~lib/memory/__stack_pointer
    i32.const 4
    i32.add
    global.set $~lib/memory/__stack_pointer
    local.get $0
    local.get $3
    i32.lt_s
    if
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store offset=16
     global.get $~lib/memory/__stack_pointer
     i32.const 8
     i32.sub
     global.set $~lib/memory/__stack_pointer
     global.get $~lib/memory/__stack_pointer
     i32.const 12392
     i32.lt_s
     br_if $folding-inner0
     global.get $~lib/memory/__stack_pointer
     i64.const 0
     i64.store
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store
     local.get $0
     local.get $2
     i32.load offset=12
     i32.ge_u
     if
      i32.const 1280
      i32.const 6384
      i32.const 114
      i32.const 42
      call $~lib/builtins/abort
      unreachable
     end
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.load offset=4
     local.get $0
     i32.const 2
     i32.shl
     i32.add
     i32.load
     local.tee $3
     i32.store offset=4
     local.get $3
     i32.eqz
     if
      i32.const 6432
      i32.const 6384
      i32.const 118
      i32.const 40
      call $~lib/builtins/abort
      unreachable
     end
     global.get $~lib/memory/__stack_pointer
     i32.const 8
     i32.add
     global.set $~lib/memory/__stack_pointer
     global.get $~lib/memory/__stack_pointer
     local.get $3
     i32.store offset=12
     global.get $~lib/memory/__stack_pointer
     i32.const 4
     i32.sub
     global.set $~lib/memory/__stack_pointer
     global.get $~lib/memory/__stack_pointer
     i32.const 12392
     i32.lt_s
     br_if $folding-inner0
     global.get $~lib/memory/__stack_pointer
     i32.const 0
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store
     local.get $1
     local.get $1
     i32.load offset=12
     local.tee $4
     i32.const 1
     i32.add
     local.tee $5
     i32.const 1
     call $~lib/array/ensureCapacity
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store
     local.get $1
     i32.load offset=4
     local.get $4
     i32.const 2
     i32.shl
     i32.add
     local.get $3
     i32.store
     local.get $1
     local.get $3
     i32.const 1
     call $~lib/rt/itcms/__link
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store
     local.get $1
     local.get $5
     i32.store offset=12
     global.get $~lib/memory/__stack_pointer
     i32.const 4
     i32.add
     global.set $~lib/memory/__stack_pointer
     local.get $0
     i32.const 1
     i32.add
     local.set $0
     br $for-loop|0
    end
   end
   global.get $~lib/memory/__stack_pointer
   local.set $2
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=16
   global.get $~lib/memory/__stack_pointer
   i32.const 6752
   i32.store offset=20
   global.get $~lib/memory/__stack_pointer
   i32.const 20
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 20
   memory.fill
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.load offset=12
   local.tee $3
   i32.const 0
   call $~lib/rt/__newArray
   local.tee $4
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $4
   i32.store
   local.get $4
   i32.load offset=4
   local.set $5
   i32.const 0
   local.set $0
   loop $for-loop|00
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store
    local.get $0
    local.get $3
    local.get $1
    i32.load offset=12
    local.tee $6
    local.get $3
    local.get $6
    i32.lt_s
    select
    i32.lt_s
    if
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=12
     global.get $~lib/memory/__stack_pointer
     local.get $0
     i32.const 2
     i32.shl
     local.tee $6
     local.get $1
     i32.load offset=4
     i32.add
     i32.load
     local.tee $7
     i32.store
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=8
     i32.const 3
     global.set $~argumentsLength
     global.get $~lib/memory/__stack_pointer
     local.get $7
     local.get $0
     local.get $1
     i32.const 6752
     i32.load
     call_indirect (type $4)
     local.tee $7
     i32.store offset=16
     local.get $5
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
     br $for-loop|00
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
   i32.const 6784
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
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
   local.set $1
   global.get $~lib/memory/__stack_pointer
   i32.const 6784
   i32.store
   local.get $0
   local.get $1
   i32.const 6784
   call $~lib/util/string/joinStringArray
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 4
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $2
   local.get $0
   i32.store offset=24
   global.get $~lib/memory/__stack_pointer
   i32.const 6656
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=12
   i32.const 6660
   local.get $0
   i32.store
   i32.const 6656
   local.get $0
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 6656
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=12
   i32.const 6656
   call $~lib/staticarray/StaticArray<~lib/string/String>#join
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 28
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 45184
  i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
   i32.const 7120
   i32.const 7184
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 7424
  i32.store offset=8
  local.get $0
  i32.const 7424
  call $~lib/string/String.__eq
  if
   global.get $~lib/memory/__stack_pointer
   local.set $0
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   i32.const 7536
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=8
   i32.const 7540
   local.get $1
   i32.store
   i32.const 7536
   local.get $1
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 7536
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=8
   local.get $0
   i32.const 7536
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
    i32.const 7680
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=24
    i32.const 7684
    local.get $2
    i32.store
    i32.const 7680
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 7680
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=24
    i32.const 7680
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
   i32.const 7712
   i32.store offset=8
   local.get $0
   i32.const 7712
   call $~lib/string/String.__eq
   if
    global.get $~lib/memory/__stack_pointer
    local.set $0
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=28
    global.get $~lib/memory/__stack_pointer
    i32.const 7888
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=8
    i32.const 7892
    local.get $1
    i32.store
    i32.const 7888
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 7888
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=8
    local.get $0
    i32.const 7888
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
     i32.const 7968
     i32.store offset=20
     global.get $~lib/memory/__stack_pointer
     local.get $2
     i32.store offset=24
     i32.const 7972
     local.get $2
     i32.store
     i32.const 7968
     local.get $2
     i32.const 1
     call $~lib/rt/itcms/__link
     global.get $~lib/memory/__stack_pointer
     i32.const 7968
     i32.store offset=20
     global.get $~lib/memory/__stack_pointer
     i32.const 1056
     i32.store offset=24
     i32.const 7968
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
    i32.const 8000
    i32.store offset=8
    local.get $0
    i32.const 8000
    call $~lib/string/String.__eq
    if
     global.get $~lib/memory/__stack_pointer
     local.set $0
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=36
     global.get $~lib/memory/__stack_pointer
     i32.const 8144
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     local.get $1
     i32.store offset=8
     i32.const 8148
     local.get $1
     i32.store
     i32.const 8144
     local.get $1
     i32.const 1
     call $~lib/rt/itcms/__link
     global.get $~lib/memory/__stack_pointer
     i32.const 8144
     i32.store offset=4
     global.get $~lib/memory/__stack_pointer
     i32.const 1056
     i32.store offset=8
     local.get $0
     i32.const 8144
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
      i32.const 8224
      i32.store offset=20
      global.get $~lib/memory/__stack_pointer
      local.get $2
      i32.store offset=24
      i32.const 8228
      local.get $2
      i32.store
      i32.const 8224
      local.get $2
      i32.const 1
      call $~lib/rt/itcms/__link
      global.get $~lib/memory/__stack_pointer
      i32.const 8224
      i32.store offset=20
      global.get $~lib/memory/__stack_pointer
      i32.const 1056
      i32.store offset=24
      i32.const 8224
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
     i32.const 8256
     i32.store offset=8
     local.get $0
     i32.const 8256
     call $~lib/string/String.__eq
     if
      global.get $~lib/memory/__stack_pointer
      local.set $0
      global.get $~lib/memory/__stack_pointer
      local.get $1
      i32.store offset=44
      global.get $~lib/memory/__stack_pointer
      i32.const 8384
      i32.store offset=4
      global.get $~lib/memory/__stack_pointer
      local.get $1
      i32.store offset=8
      i32.const 8388
      local.get $1
      i32.store
      i32.const 8384
      local.get $1
      i32.const 1
      call $~lib/rt/itcms/__link
      global.get $~lib/memory/__stack_pointer
      i32.const 8384
      i32.store offset=4
      global.get $~lib/memory/__stack_pointer
      i32.const 1056
      i32.store offset=8
      local.get $0
      i32.const 8384
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
       i32.const 8496
       i32.store offset=20
       global.get $~lib/memory/__stack_pointer
       local.get $2
       i32.store offset=24
       i32.const 8500
       local.get $2
       i32.store
       i32.const 8496
       local.get $2
       i32.const 1
       call $~lib/rt/itcms/__link
       global.get $~lib/memory/__stack_pointer
       i32.const 8496
       i32.store offset=20
       global.get $~lib/memory/__stack_pointer
       i32.const 1056
       i32.store offset=24
       i32.const 8496
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
      i32.const 8528
      i32.store offset=8
      local.get $0
      i32.const 8528
      call $~lib/string/String.__eq
      if
       global.get $~lib/memory/__stack_pointer
       local.set $0
       global.get $~lib/memory/__stack_pointer
       local.get $1
       i32.store offset=52
       global.get $~lib/memory/__stack_pointer
       i32.const 8640
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       local.get $1
       i32.store offset=8
       i32.const 8644
       local.get $1
       i32.store
       i32.const 8640
       local.get $1
       i32.const 1
       call $~lib/rt/itcms/__link
       global.get $~lib/memory/__stack_pointer
       i32.const 8640
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       i32.const 1056
       i32.store offset=8
       local.get $0
       i32.const 8640
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
        i32.const 8672
        i32.store offset=20
        global.get $~lib/memory/__stack_pointer
        local.get $2
        i32.store offset=24
        i32.const 8676
        local.get $2
        i32.store
        i32.const 8672
        local.get $2
        i32.const 1
        call $~lib/rt/itcms/__link
        global.get $~lib/memory/__stack_pointer
        i32.const 8672
        i32.store offset=20
        global.get $~lib/memory/__stack_pointer
        i32.const 1056
        i32.store offset=24
        i32.const 8672
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
       i32.const 8800
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       local.get $1
       i32.store offset=8
       i32.const 8804
       local.get $1
       i32.store
       i32.const 8800
       local.get $1
       i32.const 1
       call $~lib/rt/itcms/__link
       global.get $~lib/memory/__stack_pointer
       i32.const 8800
       i32.store offset=4
       global.get $~lib/memory/__stack_pointer
       i32.const 1056
       i32.store offset=8
       local.get $0
       i32.const 8800
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
 (func $assembly/index/performExternalInference (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 60
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.const 60
  memory.fill
  global.get $assembly/index/externalInferenceEnabled
  if (result i32)
   global.get $~lib/memory/__stack_pointer
   global.get $assembly/index/activeProvider
   local.tee $2
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=4
   local.get $2
   i32.const 1056
   call $~lib/string/String.__eq
  else
   i32.const 1
  end
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 60
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 6816
   return
  end
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/apiKeys
  local.tee $2
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $3
  i32.store offset=4
  local.get $2
  local.get $3
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#has"
  i32.eqz
  if
   global.get $~lib/memory/__stack_pointer
   i32.const 60
   i32.add
   global.set $~lib/memory/__stack_pointer
   i32.const 6976
   return
  end
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/apiKeys
  local.tee $2
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $3
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  local.get $3
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#get"
  i32.store offset=8
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/providerEndpoints
  local.tee $2
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $3
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  local.get $3
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#get"
  i32.store offset=12
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/providerModels
  local.tee $2
  i32.store
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $3
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  local.get $3
  call $"~lib/map/Map<~lib/string/String,~lib/string/String>#get"
  local.tee $3
  i32.store offset=16
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $2
  i32.store offset=20
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=24
  global.get $~lib/memory/__stack_pointer
  i32.const 7376
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=28
  i32.const 7380
  local.get $2
  i32.store
  i32.const 7376
  local.get $2
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 7376
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=28
  i32.const 7388
  local.get $3
  i32.store
  i32.const 7376
  local.get $3
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 7376
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=28
  i32.const 7376
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $2
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store
  local.get $2
  call $~lib/console/console.log
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $2
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=28
  global.get $~lib/memory/__stack_pointer
  local.get $2
  local.get $0
  local.get $1
  call $assembly/index/generateSimulatedResponse
  local.tee $1
  i32.store offset=32
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=36
  global.get $~lib/memory/__stack_pointer
  global.get $assembly/index/activeProvider
  local.tee $2
  i32.store offset=40
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=44
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
  local.tee $4
  i32.store offset=48
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.const 20
  i32.sub
  i32.load offset=16
  i32.const 3
  i32.shr_u
  call $~lib/number/I32#toString
  local.tee $5
  i32.store offset=52
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
  local.get $1
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  local.get $1
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
  i32.store offset=56
  global.get $~lib/memory/__stack_pointer
  i32.const 9440
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $1
  i32.store offset=4
  i32.const 9444
  local.get $1
  i32.store
  i32.const 9440
  local.get $1
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9440
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $2
  i32.store offset=4
  i32.const 9452
  local.get $2
  i32.store
  i32.const 9440
  local.get $2
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9440
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $3
  i32.store offset=4
  i32.const 9460
  local.get $3
  i32.store
  i32.const 9440
  local.get $3
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9440
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $4
  i32.store offset=4
  i32.const 9468
  local.get $4
  i32.store
  i32.const 9440
  local.get $4
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9440
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $5
  i32.store offset=4
  i32.const 9476
  local.get $5
  i32.store
  i32.const 9440
  local.get $5
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9440
  i32.store
  global.get $~lib/memory/__stack_pointer
  local.get $0
  i32.store offset=4
  i32.const 9484
  local.get $0
  i32.store
  i32.const 9440
  local.get $0
  i32.const 1
  call $~lib/rt/itcms/__link
  global.get $~lib/memory/__stack_pointer
  i32.const 9440
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 1056
  i32.store offset=4
  i32.const 9440
  call $~lib/staticarray/StaticArray<~lib/string/String>#join
  local.set $0
  global.get $~lib/memory/__stack_pointer
  i32.const 60
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
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 20
   memory.fill
   global.get $~lib/memory/__stack_pointer
   i32.const 2912
   i32.const 2944
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
   i32.const 12392
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
   i32.const 9936
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=16
   i32.const 9940
   local.get $0
   i32.store
   i32.const 9936
   local.get $0
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 9936
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=16
   i32.const 9948
   local.get $1
   i32.store
   i32.const 9936
   local.get $1
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 9936
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   local.get $2
   i32.store offset=16
   i32.const 9956
   local.get $2
   i32.store
   i32.const 9936
   local.get $2
   i32.const 1
   call $~lib/rt/itcms/__link
   global.get $~lib/memory/__stack_pointer
   i32.const 9936
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=16
   i32.const 9936
   call $~lib/staticarray/StaticArray<~lib/string/String>#join
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 20
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 45184
  i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
 (func $assembly/index/performChatCompletion (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 16
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 10368
   i32.store
   i32.const 10368
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
    i32.const 16
    i32.add
    global.set $~lib/memory/__stack_pointer
    i32.const 6816
    return
   end
   i32.const 1056
   local.set $2
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=8
   i32.const 1056
   local.set $3
   global.get $~lib/memory/__stack_pointer
   i32.const 1056
   i32.store offset=12
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 10496
   i32.store offset=4
   local.get $0
   i32.const 10496
   call $~lib/string/String#includes
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 10496
    i32.store offset=4
    i32.const 1
    global.set $~argumentsLength
    global.get $~lib/memory/__stack_pointer
    i32.const 8
    i32.sub
    global.set $~lib/memory/__stack_pointer
    global.get $~lib/memory/__stack_pointer
    i32.const 12392
    i32.lt_s
    br_if $folding-inner0
    global.get $~lib/memory/__stack_pointer
    i64.const 0
    i64.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 10496
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    i32.const 8
    i32.sub
    global.set $~lib/memory/__stack_pointer
    global.get $~lib/memory/__stack_pointer
    i32.const 12392
    i32.lt_s
    br_if $folding-inner0
    global.get $~lib/memory/__stack_pointer
    i64.const 0
    i64.store
    global.get $~lib/memory/__stack_pointer
    i32.const 10496
    i32.store
    block $__inlined_func$~lib/string/String#lastIndexOf$342
     i32.const 10492
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
      br $__inlined_func$~lib/string/String#lastIndexOf$342
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
      br $__inlined_func$~lib/string/String#lastIndexOf$342
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
       i32.const 10496
       i32.store offset=4
       local.get $0
       local.get $1
       i32.const 10496
       local.get $4
       call $~lib/util/string/compareImpl
       i32.eqz
       if
        global.get $~lib/memory/__stack_pointer
        i32.const 8
        i32.add
        global.set $~lib/memory/__stack_pointer
        br $__inlined_func$~lib/string/String#lastIndexOf$342
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
    i32.const 10544
    i32.store offset=4
    local.get $0
    i32.const 10544
    local.get $1
    call $~lib/string/String#indexOf
    i32.const 11
    i32.add
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 6688
    i32.store offset=4
    local.get $0
    i32.const 6688
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
     i32.store offset=8
    end
   end
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 10592
   i32.store offset=4
   local.get $0
   i32.const 10592
   call $~lib/string/String#includes
   if
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 10592
    i32.store offset=4
    local.get $0
    i32.const 10592
    i32.const 0
    call $~lib/string/String#indexOf
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 10544
    i32.store offset=4
    local.get $0
    i32.const 10544
    local.get $1
    call $~lib/string/String#indexOf
    i32.const 11
    i32.add
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 6688
    i32.store offset=4
    local.get $0
    i32.const 6688
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
     i32.store offset=12
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
    i32.const 16
    i32.add
    global.set $~lib/memory/__stack_pointer
    i32.const 10656
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
   call $assembly/index/performExternalInference
   local.set $0
   global.get $~lib/memory/__stack_pointer
   i32.const 16
   i32.add
   global.set $~lib/memory/__stack_pointer
   local.get $0
   return
  end
  i32.const 45184
  i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
   i32.const 1
   i32.const 1
   call $~lib/builtins/abort
   unreachable
  end
  global.get $~lib/memory/__stack_pointer
  i32.const 0
  i32.store
  global.get $~lib/memory/__stack_pointer
  i32.const 12096
  i32.store
  i32.const 12096
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
   i32.const 12392
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
   i32.const 12392
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
  i32.const 45184
  i32.const 45232
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/executeAgent (param $0 i32) (param $1 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 20
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 20
   memory.fill
   block $__inlined_func$assembly/index/executeAgent$324
    global.get $assembly/index/agentInitialized
    i32.eqz
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 20
     i32.add
     global.set $~lib/memory/__stack_pointer
     i32.const 1760
     local.set $0
     br $__inlined_func$assembly/index/executeAgent$324
    end
    global.get $~lib/memory/__stack_pointer
    i32.const 1856
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=8
    i32.const 1856
    local.get $0
    call $~lib/string/String#concat
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store
    local.get $1
    call $~lib/console/console.log
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/agentId
    local.tee $1
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    i32.const 2144
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    i32.const 2148
    local.get $0
    i32.store
    i32.const 2144
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2144
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=4
    i32.const 2156
    local.get $1
    i32.store
    i32.const 2144
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2144
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=4
    i32.const 2144
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $0
    global.get $~lib/memory/__stack_pointer
    i32.const 20
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
  i32.const 45184
  i32.const 45232
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/executeAgentTool (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 12
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
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
   i32.const 28
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 28
   memory.fill
   block $__inlined_func$assembly/index/executeAgentTool$325
    global.get $assembly/index/agentInitialized
    i32.eqz
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 28
     i32.add
     global.set $~lib/memory/__stack_pointer
     i32.const 1760
     local.set $0
     br $__inlined_func$assembly/index/executeAgentTool$325
    end
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=8
    global.get $~lib/memory/__stack_pointer
    i32.const 2320
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=16
    i32.const 2324
    local.get $0
    i32.store
    i32.const 2320
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2320
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=16
    i32.const 2332
    local.get $1
    i32.store
    i32.const 2320
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2320
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=16
    i32.const 2320
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
    local.get $1
    i32.store offset=24
    global.get $~lib/memory/__stack_pointer
    i32.const 2576
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=12
    i32.const 2580
    local.get $0
    i32.store
    i32.const 2576
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2576
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=12
    i32.const 2588
    local.get $1
    i32.store
    i32.const 2576
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 2576
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=12
    i32.const 2576
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $0
    global.get $~lib/memory/__stack_pointer
    i32.const 28
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
  i32.const 45184
  i32.const 45232
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
   i32.const 12392
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
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 2624
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 2624
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
  i32.const 45184
  i32.const 45232
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
   i32.const 12392
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
   i32.const 12392
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
   i32.const 2976
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=8
   i32.const 2976
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
  i32.const 45184
  i32.const 45232
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/runModelInference (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 24
   i32.sub
   global.set $~lib/memory/__stack_pointer
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.const 24
   memory.fill
   block $__inlined_func$assembly/index/runModelInference$328
    global.get $assembly/index/modelLoaded
    i32.eqz
    if
     global.get $~lib/memory/__stack_pointer
     i32.const 24
     i32.add
     global.set $~lib/memory/__stack_pointer
     i32.const 5040
     local.set $0
     br $__inlined_func$assembly/index/runModelInference$328
    end
    global.get $~lib/memory/__stack_pointer
    i32.const 5120
    i32.store offset=4
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/modelType
    local.tee $1
    i32.store offset=8
    i32.const 5120
    local.get $1
    call $~lib/string/String#concat
    local.set $1
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store
    local.get $1
    call $~lib/console/console.log
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/modelType
    local.tee $1
    i32.store offset=12
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=16
    global.get $~lib/memory/__stack_pointer
    global.get $assembly/index/modelType
    local.tee $2
    i32.store offset=20
    global.get $~lib/memory/__stack_pointer
    i32.const 5440
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $1
    i32.store offset=4
    i32.const 5444
    local.get $1
    i32.store
    i32.const 5440
    local.get $1
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5440
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $0
    i32.store offset=4
    i32.const 5452
    local.get $0
    i32.store
    i32.const 5440
    local.get $0
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5440
    i32.store
    global.get $~lib/memory/__stack_pointer
    local.get $2
    i32.store offset=4
    i32.const 5460
    local.get $2
    i32.store
    i32.const 5440
    local.get $2
    i32.const 1
    call $~lib/rt/itcms/__link
    global.get $~lib/memory/__stack_pointer
    i32.const 5440
    i32.store
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
    i32.store offset=4
    i32.const 5440
    call $~lib/staticarray/StaticArray<~lib/string/String>#join
    local.set $0
    global.get $~lib/memory/__stack_pointer
    i32.const 24
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
  i32.const 45184
  i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
 (func $export:assembly/index/performExternalInference@varargs (param $0 i32) (param $1 i32) (param $2 i32) (param $3 f32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 8
  i32.sub
  global.set $~lib/memory/__stack_pointer
  block $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i32.const 12392
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
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i32.const 0
   i32.store offset=8
   block $3of3
    block $0of3
     block $outOfRange
      global.get $~argumentsLength
      i32.const 1
      i32.sub
      br_table $0of3 $3of3 $3of3 $3of3 $outOfRange
     end
     unreachable
    end
    i32.const 1056
    local.set $1
    global.get $~lib/memory/__stack_pointer
    i32.const 1056
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
  i32.const 45184
  i32.const 45232
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
   i32.const 12392
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
   i32.const 12392
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
    i32.const 10336
    local.set $1
    global.get $~lib/memory/__stack_pointer
    i32.const 10336
    i32.store
   end
   global.get $~lib/memory/__stack_pointer
   local.get $0
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   local.get $1
   i32.store offset=8
   local.get $0
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
  i32.const 45184
  i32.const 45232
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
   i32.const 12392
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
   i32.const 12392
   i32.lt_s
   br_if $folding-inner0
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store
   global.get $~lib/memory/__stack_pointer
   i64.const 0
   i64.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 10816
   i32.store
   i32.const 10816
   call $~lib/console/console.log
   global.get $~lib/memory/__stack_pointer
   i32.const 7424
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 10960
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 11024
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 11152
   i32.store offset=12
   i32.const 7424
   i32.const 10960
   i32.const 11024
   i32.const 11152
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 7712
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 11200
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 11264
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 11344
   i32.store offset=12
   i32.const 7712
   i32.const 11200
   i32.const 11264
   i32.const 11344
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 8000
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 11392
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 11456
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 11536
   i32.store offset=12
   i32.const 8000
   i32.const 11392
   i32.const 11456
   i32.const 11536
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 8256
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 11584
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 11648
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 11728
   i32.store offset=12
   i32.const 8256
   i32.const 11584
   i32.const 11648
   i32.const 11728
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 8528
   i32.store
   global.get $~lib/memory/__stack_pointer
   i32.const 11792
   i32.store offset=4
   global.get $~lib/memory/__stack_pointer
   i32.const 11856
   i32.store offset=8
   global.get $~lib/memory/__stack_pointer
   i32.const 11936
   i32.store offset=12
   i32.const 8528
   i32.const 11792
   i32.const 11856
   i32.const 11936
   call $assembly/index/configureExternalInference
   global.get $~lib/memory/__stack_pointer
   i32.const 7712
   i32.store
   i32.const 7712
   call $assembly/index/setActiveInferenceProvider
   drop
   global.get $~lib/memory/__stack_pointer
   i32.const 11968
   i32.store
   i32.const 11968
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
  i32.const 45184
  i32.const 45232
  i32.const 1
  i32.const 1
  call $~lib/builtins/abort
  unreachable
 )
 (func $export:assembly/index/allocateString (param $0 i32) (result i32)
  global.get $~lib/memory/__stack_pointer
  i32.const 4
  i32.sub
  global.set $~lib/memory/__stack_pointer
  global.get $~lib/memory/__stack_pointer
  i32.const 12392
  i32.lt_s
  if
   i32.const 45184
   i32.const 45232
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
  local.get $0
 )
)
