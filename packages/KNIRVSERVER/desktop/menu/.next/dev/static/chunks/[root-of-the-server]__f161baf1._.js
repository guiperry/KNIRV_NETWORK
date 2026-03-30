(globalThis.TURBOPACK || (globalThis.TURBOPACK = [])).push([typeof document === "object" ? document.currentScript : undefined,
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "default",
    ()=>StarSupernova
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/jsx-dev-runtime.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/index.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/framer-motion/dist/es/render/components/motion/proxy.mjs [app-client] (ecmascript)");
;
var _s = __turbopack_context__.k.signature();
"use client";
;
;
function StarSupernova({ onComplete }) {
    _s();
    const [phase, setPhase] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])('pulsing');
    const [particles, setParticles] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])([]);
    const [menuScale, setMenuScale] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])(0);
    (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useEffect"])({
        "StarSupernova.useEffect": ()=>{
            // Start supernova after 2 seconds of pulsing
            const supernovaTimer = setTimeout({
                "StarSupernova.useEffect.supernovaTimer": ()=>{
                    setPhase('supernova');
                    // Generate particles for supernova burst with menu formation targets
                    const newParticles = [];
                    const particleCount = 200;
                    for(let i = 0; i < particleCount; i++){
                        const angle = Math.PI * 2 * i / particleCount + (Math.random() - 0.5) * 0.5;
                        const velocity = 1.5 + Math.random() * 3;
                        const size = Math.random() * 2.5 + 0.5;
                        const colorVariation = Math.random();
                        let color = "rgba(0, 200, 255, ";
                        if (colorVariation > 0.7) {
                            color = "rgba(100, 150, 255, ";
                        } else if (colorVariation > 0.4) {
                            color = "rgba(0, 220, 255, ";
                        }
                        // Calculate target positions for menu formation (ring-like structure)
                        const formationAngle = Math.PI * 2 * i / particleCount;
                        const formationRadius = 150 + Math.random() * 200;
                        const targetX = Math.cos(formationAngle) * formationRadius;
                        const targetY = Math.sin(formationAngle) * formationRadius;
                        newParticles.push({
                            id: i,
                            x: 0,
                            y: 0,
                            vx: Math.cos(angle) * velocity,
                            vy: Math.sin(angle) * velocity,
                            size,
                            opacity: 1,
                            color,
                            targetX,
                            targetY
                        });
                    }
                    setParticles(newParticles);
                    // Start forming menu after particles burst outward
                    setTimeout({
                        "StarSupernova.useEffect.supernovaTimer": ()=>{
                            setPhase('forming');
                            setMenuScale(1);
                            setTimeout(onComplete, 1500);
                        }
                    }["StarSupernova.useEffect.supernovaTimer"], 800);
                }
            }["StarSupernova.useEffect.supernovaTimer"], 2000);
            return ({
                "StarSupernova.useEffect": ()=>clearTimeout(supernovaTimer)
            })["StarSupernova.useEffect"];
        }
    }["StarSupernova.useEffect"], [
        onComplete
    ]);
    if (phase === 'complete') {
        return null;
    }
    return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        className: "fixed inset-0 z-50 flex items-center justify-center overflow-hidden",
        children: [
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                className: "absolute inset-0",
                style: {
                    backgroundColor: "#030a18"
                },
                children: [
                    ...Array(100)
                ].map((_, i)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                        className: "absolute rounded-full bg-white",
                        style: {
                            left: `${Math.random() * 100}%`,
                            top: `${Math.random() * 100}%`,
                            width: Math.random() > 0.8 ? 2 : 1,
                            height: Math.random() > 0.8 ? 2 : 1,
                            opacity: Math.random() * 0.4 + 0.1
                        },
                        animate: {
                            opacity: [
                                0.1,
                                0.4,
                                0.1
                            ]
                        },
                        transition: {
                            duration: Math.random() * 3 + 2,
                            repeat: Infinity,
                            ease: "easeInOut",
                            delay: Math.random() * 2
                        }
                    }, i, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                        lineNumber: 92,
                        columnNumber: 11
                    }, this))
            }, void 0, false, {
                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                lineNumber: 87,
                columnNumber: 7
            }, this),
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                className: "relative",
                style: {
                    width: 100,
                    height: 100
                },
                animate: phase === 'pulsing' ? {
                    scale: [
                        1,
                        1.2,
                        1
                    ]
                } : phase === 'supernova' ? {
                    scale: [
                        1,
                        3,
                        0
                    ],
                    opacity: [
                        1,
                        0.8,
                        0
                    ]
                } : {},
                transition: phase === 'pulsing' ? {
                    duration: 2,
                    repeat: Infinity,
                    ease: "easeInOut"
                } : {
                    duration: 1,
                    ease: "easeOut"
                },
                children: [
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                        className: "absolute inset-0 rounded-full",
                        style: {
                            background: `
              radial-gradient(circle at 30% 30%, 
                rgba(0, 255, 255, 0.9) 0%,
                rgba(0, 200, 255, 0.8) 10%,
                rgba(0, 180, 255, 0.7) 25%,
                rgba(0, 150, 255, 0.6) 50%,
                rgba(0, 100, 200, 0.4) 75%,
                transparent 100%
              )
            `,
                            boxShadow: `
              0 0 60px rgba(0, 200, 255, 0.8),
              0 0 100px rgba(0, 150, 255, 0.6),
              0 0 150px rgba(0, 100, 200, 0.4),
              inset -10px -10px 20px rgba(0, 50, 150, 0.3)
            `
                        }
                    }, void 0, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                        lineNumber: 148,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                        className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",
                        style: {
                            width: 40,
                            height: 40,
                            background: `
              radial-gradient(circle,
                rgba(0, 255, 255, 1) 0%,
                rgba(0, 220, 255, 0.9) 30%,
                rgba(0, 200, 255, 0.6) 60%,
                transparent 100%
              )
            `,
                            filter: 'blur(2px)'
                        },
                        animate: phase === 'pulsing' ? {
                            scale: [
                                1,
                                1.3,
                                1
                            ],
                            opacity: [
                                0.9,
                                1,
                                0.9
                            ]
                        } : {},
                        transition: {
                            duration: 2,
                            repeat: Infinity,
                            ease: "easeInOut"
                        }
                    }, void 0, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                        lineNumber: 171,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                        className: "absolute inset-0 rounded-full",
                        style: {
                            background: `
              radial-gradient(circle,
                transparent 40%,
                rgba(0, 150, 255, 0.1) 60%,
                rgba(0, 200, 255, 0.05) 80%,
                transparent 100%
              )
            `,
                            filter: 'blur(8px)'
                        },
                        animate: phase === 'pulsing' ? {
                            scale: [
                                1,
                                1.4,
                                1
                            ],
                            opacity: [
                                0.5,
                                0.8,
                                0.5
                            ]
                        } : {},
                        transition: {
                            duration: 2.5,
                            repeat: Infinity,
                            ease: "easeInOut"
                        }
                    }, void 0, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                        lineNumber: 202,
                        columnNumber: 9
                    }, this)
                ]
            }, void 0, true, {
                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                lineNumber: 116,
                columnNumber: 7
            }, this),
            phase === 'pulsing' && /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Fragment"], {
                children: [
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                        className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border-2",
                        style: {
                            width: 140,
                            height: 140,
                            borderColor: "rgba(0, 200, 255, 0.3)",
                            boxShadow: "0 0 20px rgba(0, 200, 255, 0.2)"
                        },
                        animate: {
                            scale: [
                                1,
                                1.6,
                                1
                            ],
                            opacity: [
                                0.3,
                                0.1,
                                0.3
                            ]
                        },
                        transition: {
                            duration: 2,
                            repeat: Infinity,
                            ease: "easeInOut"
                        }
                    }, void 0, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                        lineNumber: 234,
                        columnNumber: 11
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                        className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border",
                        style: {
                            width: 180,
                            height: 180,
                            borderColor: "rgba(0, 200, 255, 0.2)",
                            boxShadow: "0 0 30px rgba(0, 200, 255, 0.1)"
                        },
                        animate: {
                            scale: [
                                1,
                                1.4,
                                1
                            ],
                            opacity: [
                                0.2,
                                0.05,
                                0.2
                            ]
                        },
                        transition: {
                            duration: 2.5,
                            repeat: Infinity,
                            ease: "easeInOut"
                        }
                    }, void 0, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                        lineNumber: 252,
                        columnNumber: 11
                    }, this)
                ]
            }, void 0, true),
            phase !== 'pulsing' && particles.map((particle)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                    className: "absolute rounded-full",
                    style: {
                        width: particle.size,
                        height: particle.size,
                        backgroundColor: particle.color + "1)",
                        boxShadow: `0 0 ${particle.size * 3}px ${particle.color + "0.8)"}`
                    },
                    initial: {
                        x: 0,
                        y: 0,
                        opacity: 1
                    },
                    animate: phase === 'supernova' ? {
                        x: particle.vx * 150,
                        y: particle.vy * 150,
                        opacity: 0.8,
                        scale: [
                            1,
                            1.2,
                            1
                        ]
                    } : {
                        x: particle.targetX || 0,
                        y: particle.targetY || 0,
                        opacity: 0.3,
                        scale: [
                            1,
                            0.8,
                            0.6
                        ]
                    },
                    transition: phase === 'supernova' ? {
                        duration: 0.8,
                        ease: "easeOut"
                    } : {
                        duration: 1.5,
                        ease: "easeInOut"
                    }
                }, particle.id, false, {
                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                    lineNumber: 275,
                    columnNumber: 9
                }, this)),
            phase === 'forming' && /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",
                style: {
                    width: 900,
                    height: 900
                },
                animate: {
                    scale: menuScale,
                    opacity: menuScale * 0.3
                },
                transition: {
                    duration: 1.5,
                    ease: "easeOut"
                },
                children: [
                    [
                        120,
                        180,
                        250,
                        320
                    ].map((radius, i)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                            className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border",
                            style: {
                                width: radius * 2,
                                height: radius * 2,
                                borderColor: "rgba(0, 200, 255, 0.2)",
                                boxShadow: `0 0 ${20 - i * 3}px rgba(0, 200, 255, 0.1)`
                            }
                        }, i, false, {
                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                            lineNumber: 337,
                            columnNumber: 13
                        }, this)),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                        className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",
                        style: {
                            width: 160,
                            height: 160,
                            background: "rgba(0, 100, 150, 0.1)",
                            border: "2px solid rgba(0, 200, 255, 0.3)",
                            boxShadow: "0 0 30px rgba(0, 200, 255, 0.2)"
                        }
                    }, void 0, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                        lineNumber: 350,
                        columnNumber: 11
                    }, this)
                ]
            }, void 0, true, {
                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
                lineNumber: 320,
                columnNumber: 9
            }, this)
        ]
    }, void 0, true, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx",
        lineNumber: 85,
        columnNumber: 5
    }, this);
}
_s(StarSupernova, "P5F9kUOPfrMkybCx4358e9A3F+g=");
_c = StarSupernova;
var _c;
__turbopack_context__.k.register(_c, "StarSupernova");
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
"[next]/internal/font/google/inter_9810da39.module.css [app-client] (css module)", ((__turbopack_context__) => {

__turbopack_context__.v({
  "className": "inter_9810da39-module__5cOL3W__className",
});
}),
"[next]/internal/font/google/inter_9810da39.js [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "default",
    ()=>__TURBOPACK__default__export__
]);
var __TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$module$2e$css__$5b$app$2d$client$5d$__$28$css__module$29$__ = __turbopack_context__.i("[next]/internal/font/google/inter_9810da39.module.css [app-client] (css module)");
;
const fontData = {
    className: __TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$module$2e$css__$5b$app$2d$client$5d$__$28$css__module$29$__["default"].className,
    style: {
        fontFamily: "'Inter', 'Inter Fallback'",
        fontStyle: "normal"
    }
};
if (__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$module$2e$css__$5b$app$2d$client$5d$__$28$css__module$29$__["default"].variable != null) {
    fontData.variable = __TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$module$2e$css__$5b$app$2d$client$5d$__$28$css__module$29$__["default"].variable;
}
const __TURBOPACK__default__export__ = fontData;
}),
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/lib/utils.ts [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "cn",
    ()=>cn
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$clsx$2f$dist$2f$clsx$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/clsx/dist/clsx.mjs [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$tailwind$2d$merge$2f$dist$2f$bundle$2d$mjs$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/tailwind-merge/dist/bundle-mjs.mjs [app-client] (ecmascript)");
;
;
function cn(...inputs) {
    return (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$tailwind$2d$merge$2f$dist$2f$bundle$2d$mjs$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["twMerge"])((0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$clsx$2f$dist$2f$clsx$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["clsx"])(inputs));
}
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "Card",
    ()=>Card,
    "CardContent",
    ()=>CardContent,
    "CardDescription",
    ()=>CardDescription,
    "CardFooter",
    ()=>CardFooter,
    "CardHeader",
    ()=>CardHeader,
    "CardTitle",
    ()=>CardTitle
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/jsx-dev-runtime.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/index.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/lib/utils.ts [app-client] (ecmascript)");
;
;
;
const Card = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("rounded-lg border bg-card text-card-foreground shadow-sm", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx",
        lineNumber: 9,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c1 = Card;
Card.displayName = "Card";
const CardHeader = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c2 = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("flex flex-col space-y-1.5 p-6", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx",
        lineNumber: 24,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c3 = CardHeader;
CardHeader.displayName = "CardHeader";
const CardTitle = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c4 = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("h3", {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("text-2xl font-semibold leading-none tracking-tight", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx",
        lineNumber: 32,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c5 = CardTitle;
CardTitle.displayName = "CardTitle";
const CardDescription = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c6 = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("p", {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("text-sm text-muted-foreground", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx",
        lineNumber: 47,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c7 = CardDescription;
CardDescription.displayName = "CardDescription";
const CardContent = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c8 = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("p-6 pt-0", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx",
        lineNumber: 59,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c9 = CardContent;
CardContent.displayName = "CardContent";
const CardFooter = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c10 = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("flex items-center p-6 pt-0", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx",
        lineNumber: 67,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c11 = CardFooter;
CardFooter.displayName = "CardFooter";
;
var _c, _c1, _c2, _c3, _c4, _c5, _c6, _c7, _c8, _c9, _c10, _c11;
__turbopack_context__.k.register(_c, "Card$React.forwardRef");
__turbopack_context__.k.register(_c1, "Card");
__turbopack_context__.k.register(_c2, "CardHeader$React.forwardRef");
__turbopack_context__.k.register(_c3, "CardHeader");
__turbopack_context__.k.register(_c4, "CardTitle$React.forwardRef");
__turbopack_context__.k.register(_c5, "CardTitle");
__turbopack_context__.k.register(_c6, "CardDescription$React.forwardRef");
__turbopack_context__.k.register(_c7, "CardDescription");
__turbopack_context__.k.register(_c8, "CardContent$React.forwardRef");
__turbopack_context__.k.register(_c9, "CardContent");
__turbopack_context__.k.register(_c10, "CardFooter$React.forwardRef");
__turbopack_context__.k.register(_c11, "CardFooter");
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/button.tsx [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "Button",
    ()=>Button,
    "buttonVariants",
    ()=>buttonVariants
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/jsx-dev-runtime.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/index.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$slot$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/@radix-ui/react-slot/dist/index.mjs [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$class$2d$variance$2d$authority$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/class-variance-authority/dist/index.mjs [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/lib/utils.ts [app-client] (ecmascript)");
;
;
;
;
;
const buttonVariants = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$class$2d$variance$2d$authority$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cva"])("inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0", {
    variants: {
        variant: {
            default: "bg-primary text-primary-foreground shadow hover:bg-primary/90",
            destructive: "bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90",
            outline: "border border-input bg-background shadow-sm hover:bg-accent hover:text-accent-foreground",
            secondary: "bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80",
            ghost: "hover:bg-accent hover:text-accent-foreground",
            link: "text-primary underline-offset-4 hover:underline"
        },
        size: {
            default: "h-9 px-4 py-2",
            sm: "h-8 rounded-md px-3 text-xs",
            lg: "h-10 rounded-md px-8",
            icon: "h-9 w-9"
        }
    },
    defaultVariants: {
        variant: "default",
        size: "default"
    }
});
const Button = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c = ({ className, variant, size, asChild = false, ...props }, ref)=>{
    const Comp = asChild ? __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$slot$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Slot"] : "button";
    return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(Comp, {
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])(buttonVariants({
            variant,
            size,
            className
        })),
        ref: ref,
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/button.tsx",
        lineNumber: 47,
        columnNumber: 7
    }, ("TURBOPACK compile-time value", void 0));
});
_c1 = Button;
Button.displayName = "Button";
;
var _c, _c1;
__turbopack_context__.k.register(_c, "Button$React.forwardRef");
__turbopack_context__.k.register(_c1, "Button");
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/switch.tsx [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "Switch",
    ()=>Switch
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/jsx-dev-runtime.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/index.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$switch$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/@radix-ui/react-switch/dist/index.mjs [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/lib/utils.ts [app-client] (ecmascript)");
;
;
;
;
const Switch = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$switch$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Root"], {
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-cyan-500 data-[state=unchecked]:bg-gray-600", className),
        ...props,
        ref: ref,
        children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$switch$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Thumb"], {
            className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("pointer-events-none block h-4 w-4 rounded-full bg-white shadow-lg ring-0 transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0")
        }, void 0, false, {
            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/switch.tsx",
            lineNumber: 18,
            columnNumber: 5
        }, ("TURBOPACK compile-time value", void 0))
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/switch.tsx",
        lineNumber: 10,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c1 = Switch;
Switch.displayName = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$switch$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Root"].displayName;
;
var _c, _c1;
__turbopack_context__.k.register(_c, "Switch$React.forwardRef");
__turbopack_context__.k.register(_c1, "Switch");
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "Dialog",
    ()=>Dialog,
    "DialogClose",
    ()=>DialogClose,
    "DialogContent",
    ()=>DialogContent,
    "DialogDescription",
    ()=>DialogDescription,
    "DialogFooter",
    ()=>DialogFooter,
    "DialogHeader",
    ()=>DialogHeader,
    "DialogOverlay",
    ()=>DialogOverlay,
    "DialogPortal",
    ()=>DialogPortal,
    "DialogTitle",
    ()=>DialogTitle,
    "DialogTrigger",
    ()=>DialogTrigger
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/jsx-dev-runtime.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/index.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/@radix-ui/react-dialog/dist/index.mjs [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$x$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__X$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/x.js [app-client] (ecmascript) <export default as X>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/lib/utils.ts [app-client] (ecmascript)");
;
;
;
;
;
const Dialog = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Root"];
const DialogTrigger = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Trigger"];
const DialogPortal = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Portal"];
const DialogClose = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Close"];
const DialogOverlay = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Overlay"], {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("fixed inset-0 z-50 bg-black/80  data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
        lineNumber: 19,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c = DialogOverlay;
DialogOverlay.displayName = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Overlay"].displayName;
const DialogContent = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c1 = ({ className, children, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(DialogPortal, {
        children: [
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(DialogOverlay, {}, void 0, false, {
                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
                lineNumber: 35,
                columnNumber: 5
            }, ("TURBOPACK compile-time value", void 0)),
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Content"], {
                ref: ref,
                className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:rounded-lg", className),
                ...props,
                children: [
                    children,
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Close"], {
                        className: "absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground",
                        children: [
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$x$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__X$3e$__["X"], {
                                className: "h-4 w-4"
                            }, void 0, false, {
                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
                                lineNumber: 46,
                                columnNumber: 9
                            }, ("TURBOPACK compile-time value", void 0)),
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("span", {
                                className: "sr-only",
                                children: "Close"
                            }, void 0, false, {
                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
                                lineNumber: 47,
                                columnNumber: 9
                            }, ("TURBOPACK compile-time value", void 0))
                        ]
                    }, void 0, true, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
                        lineNumber: 45,
                        columnNumber: 7
                    }, ("TURBOPACK compile-time value", void 0))
                ]
            }, void 0, true, {
                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
                lineNumber: 36,
                columnNumber: 5
            }, ("TURBOPACK compile-time value", void 0))
        ]
    }, void 0, true, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
        lineNumber: 34,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c2 = DialogContent;
DialogContent.displayName = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Content"].displayName;
const DialogHeader = ({ className, ...props })=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("flex flex-col space-y-1.5 text-center sm:text-left", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
        lineNumber: 58,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0));
_c3 = DialogHeader;
DialogHeader.displayName = "DialogHeader";
const DialogFooter = ({ className, ...props })=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
        lineNumber: 72,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0));
_c4 = DialogFooter;
DialogFooter.displayName = "DialogFooter";
const DialogTitle = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c5 = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Title"], {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("text-lg font-semibold leading-none tracking-tight", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
        lineNumber: 86,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c6 = DialogTitle;
DialogTitle.displayName = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Title"].displayName;
const DialogDescription = /*#__PURE__*/ __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["forwardRef"](_c7 = ({ className, ...props }, ref)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Description"], {
        ref: ref,
        className: (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$lib$2f$utils$2e$ts__$5b$app$2d$client$5d$__$28$ecmascript$29$__["cn"])("text-sm text-muted-foreground", className),
        ...props
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx",
        lineNumber: 101,
        columnNumber: 3
    }, ("TURBOPACK compile-time value", void 0)));
_c8 = DialogDescription;
DialogDescription.displayName = __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f40$radix$2d$ui$2f$react$2d$dialog$2f$dist$2f$index$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Description"].displayName;
;
var _c, _c1, _c2, _c3, _c4, _c5, _c6, _c7, _c8;
__turbopack_context__.k.register(_c, "DialogOverlay");
__turbopack_context__.k.register(_c1, "DialogContent$React.forwardRef");
__turbopack_context__.k.register(_c2, "DialogContent");
__turbopack_context__.k.register(_c3, "DialogHeader");
__turbopack_context__.k.register(_c4, "DialogFooter");
__turbopack_context__.k.register(_c5, "DialogTitle$React.forwardRef");
__turbopack_context__.k.register(_c6, "DialogTitle");
__turbopack_context__.k.register(_c7, "DialogDescription$React.forwardRef");
__turbopack_context__.k.register(_c8, "DialogDescription");
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "default",
    ()=>__TURBOPACK__default__export__
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/jsx-dev-runtime.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/index.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[next]/internal/font/google/inter_9810da39.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/card.tsx [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$button$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/button.tsx [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$switch$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/switch.tsx [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$dialog$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/ui/dialog.tsx [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$volume$2d$2$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Volume2$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/volume-2.js [app-client] (ecmascript) <export default as Volume2>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$palette$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Palette$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/palette.js [app-client] (ecmascript) <export default as Palette>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$zap$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Zap$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/zap.js [app-client] (ecmascript) <export default as Zap>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$shield$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Shield$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/shield.js [app-client] (ecmascript) <export default as Shield>");
;
var _s = __turbopack_context__.k.signature();
"use client";
;
;
;
;
;
;
;
const defaultSettings = {
    soundEnabled: true,
    animationsEnabled: true,
    darkMode: true,
    language: "en",
    performanceMode: false,
    securityMode: true
};
const SettingsModal = ({ isOpen, onClose })=>{
    _s();
    const [settings, setSettings] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])(defaultSettings);
    (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useEffect"])({
        "SettingsModal.useEffect": ()=>{
            const handleEscape = {
                "SettingsModal.useEffect.handleEscape": (e)=>{
                    if (e.key === 'Escape') {
                        onClose();
                    }
                }
            }["SettingsModal.useEffect.handleEscape"];
            if (isOpen) {
                document.addEventListener('keydown', handleEscape);
            }
            return ({
                "SettingsModal.useEffect": ()=>{
                    document.removeEventListener('keydown', handleEscape);
                }
            })["SettingsModal.useEffect"];
        }
    }["SettingsModal.useEffect"], [
        isOpen,
        onClose
    ]);
    const handleSettingChange = (key, value)=>{
        setSettings((prev)=>({
                ...prev,
                [key]: value
            }));
    };
    const resetSettings = ()=>{
        setSettings(defaultSettings);
    };
    const saveSettings = ()=>{
        // Here you would typically save to localStorage or backend
        console.log('Saving settings:', settings);
        onClose();
    };
    if (!isOpen) return null;
    return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$dialog$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Dialog"], {
        open: isOpen,
        onOpenChange: onClose,
        children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$dialog$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["DialogContent"], {
            className: `w-full max-w-2xl mx-4 bg-gray-900/95 border-cyan-500/30 text-white ${__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"].className}`,
            children: [
                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$dialog$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["DialogHeader"], {
                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$dialog$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["DialogTitle"], {
                        className: `text-cyan-400 text-xl font-bold flex items-center gap-2 ${__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"].className}`,
                        children: [
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$shield$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Shield$3e$__["Shield"], {
                                className: "h-5 w-5"
                            }, void 0, false, {
                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                lineNumber: 79,
                                columnNumber: 13
                            }, ("TURBOPACK compile-time value", void 0)),
                            "Settings"
                        ]
                    }, void 0, true, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                        lineNumber: 78,
                        columnNumber: 11
                    }, ("TURBOPACK compile-time value", void 0))
                }, void 0, false, {
                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                    lineNumber: 77,
                    columnNumber: 9
                }, ("TURBOPACK compile-time value", void 0)),
                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardContent"], {
                    className: "space-y-6",
                    children: [
                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Card"], {
                            className: "bg-gray-800/50 border-cyan-500/20",
                            children: [
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardHeader"], {
                                    className: "pb-3",
                                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardTitle"], {
                                        className: "text-cyan-300 text-base flex items-center gap-2",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$volume$2d$2$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Volume2$3e$__["Volume2"], {
                                                className: "h-4 w-4"
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 89,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0)),
                                            "Audio"
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                        lineNumber: 88,
                                        columnNumber: 15
                                    }, ("TURBOPACK compile-time value", void 0))
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 87,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0)),
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardContent"], {
                                    className: "space-y-3",
                                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "flex items-center justify-between",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("label", {
                                                className: `text-sm text-gray-300 ${__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"].className}`,
                                                children: "Sound Effects"
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 95,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0)),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$switch$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Switch"], {
                                                checked: settings.soundEnabled,
                                                onCheckedChange: (checked)=>handleSettingChange('soundEnabled', checked)
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 96,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0))
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                        lineNumber: 94,
                                        columnNumber: 15
                                    }, ("TURBOPACK compile-time value", void 0))
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 93,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0))
                            ]
                        }, void 0, true, {
                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                            lineNumber: 86,
                            columnNumber: 11
                        }, ("TURBOPACK compile-time value", void 0)),
                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Card"], {
                            className: "bg-gray-800/50 border-cyan-500/20",
                            children: [
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardHeader"], {
                                    className: "pb-3",
                                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardTitle"], {
                                        className: "text-cyan-300 text-base flex items-center gap-2",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$palette$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Palette$3e$__["Palette"], {
                                                className: "h-4 w-4"
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 108,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0)),
                                            "Visual"
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                        lineNumber: 107,
                                        columnNumber: 15
                                    }, ("TURBOPACK compile-time value", void 0))
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 106,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0)),
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardContent"], {
                                    className: "space-y-3",
                                    children: [
                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                            className: "flex items-center justify-between",
                                            children: [
                                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("label", {
                                                    className: `text-sm text-gray-300 ${__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"].className}`,
                                                    children: "Animations"
                                                }, void 0, false, {
                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                    lineNumber: 114,
                                                    columnNumber: 17
                                                }, ("TURBOPACK compile-time value", void 0)),
                                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$switch$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Switch"], {
                                                    checked: settings.animationsEnabled,
                                                    onCheckedChange: (checked)=>handleSettingChange('animationsEnabled', checked)
                                                }, void 0, false, {
                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                    lineNumber: 115,
                                                    columnNumber: 17
                                                }, ("TURBOPACK compile-time value", void 0))
                                            ]
                                        }, void 0, true, {
                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                            lineNumber: 113,
                                            columnNumber: 15
                                        }, ("TURBOPACK compile-time value", void 0)),
                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                            className: "flex items-center justify-between",
                                            children: [
                                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("label", {
                                                    className: `text-sm text-gray-300 ${__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"].className}`,
                                                    children: "Dark Mode"
                                                }, void 0, false, {
                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                    lineNumber: 121,
                                                    columnNumber: 17
                                                }, ("TURBOPACK compile-time value", void 0)),
                                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$switch$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Switch"], {
                                                    checked: settings.darkMode,
                                                    onCheckedChange: (checked)=>handleSettingChange('darkMode', checked)
                                                }, void 0, false, {
                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                    lineNumber: 122,
                                                    columnNumber: 17
                                                }, ("TURBOPACK compile-time value", void 0))
                                            ]
                                        }, void 0, true, {
                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                            lineNumber: 120,
                                            columnNumber: 15
                                        }, ("TURBOPACK compile-time value", void 0))
                                    ]
                                }, void 0, true, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 112,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0))
                            ]
                        }, void 0, true, {
                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                            lineNumber: 105,
                            columnNumber: 11
                        }, ("TURBOPACK compile-time value", void 0)),
                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Card"], {
                            className: "bg-gray-800/50 border-cyan-500/20",
                            children: [
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardHeader"], {
                                    className: "pb-3",
                                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardTitle"], {
                                        className: "text-cyan-300 text-base flex items-center gap-2",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$zap$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Zap$3e$__["Zap"], {
                                                className: "h-4 w-4"
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 134,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0)),
                                            "Performance"
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                        lineNumber: 133,
                                        columnNumber: 15
                                    }, ("TURBOPACK compile-time value", void 0))
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 132,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0)),
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardContent"], {
                                    className: "space-y-3",
                                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "flex items-center justify-between",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("label", {
                                                className: `text-sm text-gray-300 ${__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"].className}`,
                                                children: "Performance Mode"
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 140,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0)),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$switch$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Switch"], {
                                                checked: settings.performanceMode,
                                                onCheckedChange: (checked)=>handleSettingChange('performanceMode', checked)
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 141,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0))
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                        lineNumber: 139,
                                        columnNumber: 15
                                    }, ("TURBOPACK compile-time value", void 0))
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 138,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0))
                            ]
                        }, void 0, true, {
                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                            lineNumber: 131,
                            columnNumber: 11
                        }, ("TURBOPACK compile-time value", void 0)),
                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Card"], {
                            className: "bg-gray-800/50 border-cyan-500/20",
                            children: [
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardHeader"], {
                                    className: "pb-3",
                                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardTitle"], {
                                        className: "text-cyan-300 text-base flex items-center gap-2",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$shield$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Shield$3e$__["Shield"], {
                                                className: "h-4 w-4"
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 153,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0)),
                                            "Security"
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                        lineNumber: 152,
                                        columnNumber: 15
                                    }, ("TURBOPACK compile-time value", void 0))
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 151,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0)),
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$card$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["CardContent"], {
                                    className: "space-y-3",
                                    children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "flex items-center justify-between",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("label", {
                                                className: `text-sm text-gray-300 ${__TURBOPACK__imported__module__$5b$next$5d2f$internal$2f$font$2f$google$2f$inter_9810da39$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"].className}`,
                                                children: "Enhanced Security"
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 159,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0)),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$switch$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Switch"], {
                                                checked: settings.securityMode,
                                                onCheckedChange: (checked)=>handleSettingChange('securityMode', checked)
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                                lineNumber: 160,
                                                columnNumber: 17
                                            }, ("TURBOPACK compile-time value", void 0))
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                        lineNumber: 158,
                                        columnNumber: 15
                                    }, ("TURBOPACK compile-time value", void 0))
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 157,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0))
                            ]
                        }, void 0, true, {
                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                            lineNumber: 150,
                            columnNumber: 11
                        }, ("TURBOPACK compile-time value", void 0)),
                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                            className: "flex justify-between pt-4 border-t border-gray-700",
                            children: [
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$button$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Button"], {
                                    variant: "outline",
                                    onClick: resetSettings,
                                    className: "border-cyan-500/30 text-cyan-400 hover:bg-cyan-500/10",
                                    children: "Reset to Default"
                                }, void 0, false, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 170,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0)),
                                /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                    className: "flex space-x-2",
                                    children: [
                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$button$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Button"], {
                                            variant: "outline",
                                            onClick: onClose,
                                            className: "border-cyan-500/30 text-cyan-400 hover:bg-cyan-500/10",
                                            children: "Cancel"
                                        }, void 0, false, {
                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                            lineNumber: 178,
                                            columnNumber: 15
                                        }, ("TURBOPACK compile-time value", void 0)),
                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$ui$2f$button$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Button"], {
                                            onClick: saveSettings,
                                            className: "bg-cyan-500 hover:bg-cyan-600 text-black font-semibold",
                                            children: "Save Settings"
                                        }, void 0, false, {
                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                            lineNumber: 185,
                                            columnNumber: 15
                                        }, ("TURBOPACK compile-time value", void 0))
                                    ]
                                }, void 0, true, {
                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                                    lineNumber: 177,
                                    columnNumber: 13
                                }, ("TURBOPACK compile-time value", void 0))
                            ]
                        }, void 0, true, {
                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                            lineNumber: 169,
                            columnNumber: 11
                        }, ("TURBOPACK compile-time value", void 0))
                    ]
                }, void 0, true, {
                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
                    lineNumber: 84,
                    columnNumber: 9
                }, ("TURBOPACK compile-time value", void 0))
            ]
        }, void 0, true, {
            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
            lineNumber: 76,
            columnNumber: 7
        }, ("TURBOPACK compile-time value", void 0))
    }, void 0, false, {
        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx",
        lineNumber: 75,
        columnNumber: 5
    }, ("TURBOPACK compile-time value", void 0));
};
_s(SettingsModal, "ZGre7If7w4HenpmZpG3z+HIEEH4=");
_c = SettingsModal;
const __TURBOPACK__default__export__ = SettingsModal;
var _c;
__turbopack_context__.k.register(_c, "SettingsModal");
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
"[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx [app-client] (ecmascript)", ((__turbopack_context__) => {
"use strict";

__turbopack_context__.s([
    "default",
    ()=>ConstellationMenu
]);
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/jsx-dev-runtime.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/next/dist/compiled/react/index.js [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/framer-motion/dist/es/render/components/motion/proxy.mjs [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$eye$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Eye$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/eye.js [app-client] (ecmascript) <export default as Eye>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$box$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Box$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/box.js [app-client] (ecmascript) <export default as Box>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$wrench$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Wrench$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/wrench.js [app-client] (ecmascript) <export default as Wrench>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$globe$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Globe$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/globe.js [app-client] (ecmascript) <export default as Globe>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$cpu$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Cpu$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/cpu.js [app-client] (ecmascript) <export default as Cpu>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$layers$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Layers$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/layers.js [app-client] (ecmascript) <export default as Layers>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$settings$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Settings$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/settings.js [app-client] (ecmascript) <export default as Settings>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$triangle$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Triangle$3e$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/node_modules/lucide-react/dist/esm/icons/triangle.js [app-client] (ecmascript) <export default as Triangle>");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$star$2d$supernova$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/star-supernova.tsx [app-client] (ecmascript)");
var __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$settings$2d$modal$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__ = __turbopack_context__.i("[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/settings-modal.tsx [app-client] (ecmascript)");
;
var _s = __turbopack_context__.k.signature();
"use client";
;
;
;
;
;
// Inner ring labels (closest to center)
const innerLabels = [
    {
        label: "KNIRVCLI",
        angle: 210
    },
    {
        label: "KNIRVCTCLI",
        angle: 250
    },
    {
        label: "KNIRVCHAIN",
        angle: 290
    },
    {
        label: "KNIRVCCHAIN",
        angle: 330
    },
    {
        label: "KNIRVCINEXUS",
        angle: 10
    },
    {
        label: "KNIRVROUTER",
        angle: 50
    },
    {
        label: "KNIRVCONTROLLER",
        angle: 90
    },
    {
        label: "KNIRVCLLS",
        angle: 130
    },
    {
        label: "KNIRVCWORKER",
        angle: 170
    }
];
// Middle ring labels
const middleLabels = [
    {
        label: "KNIRVCORPAY",
        angle: 230
    },
    {
        label: "KNIRVASDK",
        angle: 270
    },
    {
        label: "KNIRVCOGINS",
        angle: 310
    },
    {
        label: "KNIRVCORTEX",
        angle: 350
    },
    {
        label: "KNIRVGATEWAY",
        angle: 30
    },
    {
        label: "KNIRVCONTROLLER",
        angle: 70
    },
    {
        label: "KNIRVROUTET",
        angle: 110
    },
    {
        label: "KNIRVBALI",
        angle: 150
    },
    {
        label: "KNIRVTESTNET",
        angle: 190
    }
];
// Outer icons positioned around the outer ring
const outerIcons = [
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$triangle$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Triangle$3e$__["Triangle"],
        angle: 0
    },
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$globe$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Globe$3e$__["Globe"],
        angle: 45
    },
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$wrench$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Wrench$3e$__["Wrench"],
        angle: 90
    },
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$cpu$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Cpu$3e$__["Cpu"],
        angle: 135
    },
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$layers$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Layers$3e$__["Layers"],
        angle: 180
    },
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$settings$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Settings$3e$__["Settings"],
        angle: 225
    },
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$eye$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Eye$3e$__["Eye"],
        angle: 270
    },
    {
        icon: __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$box$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Box$3e$__["Box"],
        angle: 315
    }
];
function polarToCartesian(angle, radius) {
    const radian = (angle - 90) * (Math.PI / 180);
    return {
        x: Math.cos(radian) * radius,
        y: Math.sin(radian) * radius
    };
}
function ConstellationMenu() {
    _s();
    const [mounted, setMounted] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])(false);
    const [hoveredIcon, setHoveredIcon] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])(null);
    const [loadingComplete, setLoadingComplete] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])(false);
    const [settingsOpen, setSettingsOpen] = (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useState"])(false);
    (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$index$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["useEffect"])({
        "ConstellationMenu.useEffect": ()=>{
            setMounted(true);
        }
    }["ConstellationMenu.useEffect"], []);
    const handleLoadingComplete = ()=>{
        setLoadingComplete(true);
    };
    const centerSize = 160;
    const ring1Radius = 120;
    const ring2Radius = 180;
    const ring3Radius = 250;
    const ring4Radius = 320;
    const iconRadius = 360 // Position icons just outside the outer ring
    ;
    return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Fragment"], {
        children: [
            !loadingComplete && /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$star$2d$supernova$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"], {
                onComplete: handleLoadingComplete
            }, void 0, false, {
                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                lineNumber: 79,
                columnNumber: 28
            }, this),
            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                className: "relative flex min-h-screen w-full items-center justify-center overflow-hidden",
                style: {
                    backgroundColor: "#030a18"
                },
                children: [
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                        className: "relative",
                        children: [
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                className: "absolute inset-0",
                                children: [
                                    ...Array(200)
                                ].map((_, i)=>/*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                                        className: "absolute rounded-full bg-white",
                                        style: {
                                            left: `${Math.random() * 100}%`,
                                            top: `${Math.random() * 100}%`,
                                            width: Math.random() > 0.9 ? 2 : 1,
                                            height: Math.random() > 0.9 ? 2 : 1,
                                            opacity: Math.random() * 0.6 + 0.2
                                        },
                                        animate: {
                                            opacity: [
                                                0.2,
                                                0.6,
                                                0.2
                                            ]
                                        },
                                        transition: {
                                            duration: Math.random() * 4 + 2,
                                            repeat: Infinity,
                                            ease: "easeInOut",
                                            delay: Math.random() * 2
                                        }
                                    }, i, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 90,
                                        columnNumber: 11
                                    }, this))
                            }, void 0, false, {
                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                lineNumber: 88,
                                columnNumber: 7
                            }, this),
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                className: "pointer-events-none absolute bottom-0 left-0 right-0 h-48",
                                children: [
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "absolute inset-0 bg-gradient-to-t from-cyan-500/10 via-cyan-500/5 to-transparent"
                                    }, void 0, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 115,
                                        columnNumber: 9
                                    }, this),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "absolute bottom-12 left-1/2 h-px w-[80%] -translate-x-1/2 bg-gradient-to-r from-transparent via-cyan-400/60 to-transparent"
                                    }, void 0, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 116,
                                        columnNumber: 9
                                    }, this),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "absolute bottom-0 left-[10%] h-16 w-4 bg-gradient-to-t from-cyan-900/30 to-transparent",
                                        style: {
                                            clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)'
                                        }
                                    }, void 0, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 118,
                                        columnNumber: 9
                                    }, this),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "absolute bottom-0 left-[15%] h-24 w-6 bg-gradient-to-t from-cyan-900/20 to-transparent",
                                        style: {
                                            clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)'
                                        }
                                    }, void 0, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 119,
                                        columnNumber: 9
                                    }, this),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "absolute bottom-0 right-[10%] h-20 w-5 bg-gradient-to-t from-cyan-900/30 to-transparent",
                                        style: {
                                            clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)'
                                        }
                                    }, void 0, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 120,
                                        columnNumber: 9
                                    }, this),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "absolute bottom-0 right-[15%] h-28 w-7 bg-gradient-to-t from-cyan-900/20 to-transparent",
                                        style: {
                                            clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)'
                                        }
                                    }, void 0, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 121,
                                        columnNumber: 9
                                    }, this)
                                ]
                            }, void 0, true, {
                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                lineNumber: 114,
                                columnNumber: 7
                            }, this),
                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                className: "relative",
                                style: {
                                    width: 900,
                                    height: 900
                                },
                                children: [
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("svg", {
                                        className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",
                                        width: "900",
                                        height: "900",
                                        viewBox: "-450 -450 900 900",
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                cx: 0,
                                                cy: 0,
                                                r: ring4Radius,
                                                fill: "none",
                                                stroke: "rgba(0, 200, 255, 0.3)",
                                                strokeWidth: "1.5",
                                                initial: {
                                                    pathLength: 0,
                                                    opacity: 0
                                                },
                                                animate: loadingComplete ? {
                                                    pathLength: 1,
                                                    opacity: 1
                                                } : {},
                                                transition: {
                                                    duration: 2,
                                                    delay: 0.2
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 137,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                cx: 0,
                                                cy: 0,
                                                r: 355,
                                                fill: "none",
                                                stroke: "rgba(0, 200, 255, 0.2)",
                                                strokeWidth: "1",
                                                initial: {
                                                    pathLength: 0,
                                                    opacity: 0
                                                },
                                                animate: loadingComplete ? {
                                                    pathLength: 1,
                                                    opacity: 1
                                                } : {},
                                                transition: {
                                                    duration: 2,
                                                    delay: 0.1
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 150,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                cx: 0,
                                                cy: 0,
                                                r: 365,
                                                fill: "none",
                                                stroke: "rgba(0, 200, 255, 0.2)",
                                                strokeWidth: "1",
                                                initial: {
                                                    pathLength: 0,
                                                    opacity: 0
                                                },
                                                animate: loadingComplete ? {
                                                    pathLength: 1,
                                                    opacity: 1
                                                } : {},
                                                transition: {
                                                    duration: 2,
                                                    delay: 0.1
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 163,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                cx: 0,
                                                cy: 0,
                                                r: ring3Radius,
                                                fill: "none",
                                                stroke: "rgba(0, 200, 255, 0.25)",
                                                strokeWidth: "1.5",
                                                initial: {
                                                    pathLength: 0,
                                                    opacity: 0
                                                },
                                                animate: loadingComplete ? {
                                                    pathLength: 1,
                                                    opacity: 1
                                                } : {},
                                                transition: {
                                                    duration: 2,
                                                    delay: 0.4
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 176,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                cx: 0,
                                                cy: 0,
                                                r: ring2Radius,
                                                fill: "none",
                                                stroke: "rgba(0, 200, 255, 0.35)",
                                                strokeWidth: "1.5",
                                                initial: {
                                                    pathLength: 0,
                                                    opacity: 0
                                                },
                                                animate: loadingComplete ? {
                                                    pathLength: 1,
                                                    opacity: 1
                                                } : {},
                                                transition: {
                                                    duration: 2,
                                                    delay: 0.6
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 189,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                cx: 0,
                                                cy: 0,
                                                r: ring1Radius,
                                                fill: "none",
                                                stroke: "rgba(0, 200, 255, 0.4)",
                                                strokeWidth: "2",
                                                initial: {
                                                    pathLength: 0,
                                                    opacity: 0
                                                },
                                                animate: loadingComplete ? {
                                                    pathLength: 1,
                                                    opacity: 1
                                                } : {},
                                                transition: {
                                                    duration: 2,
                                                    delay: 0.8
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 202,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].g, {
                                                initial: {
                                                    opacity: 0,
                                                    scale: 0.8
                                                },
                                                animate: mounted ? {
                                                    opacity: 1,
                                                    scale: 1
                                                } : {},
                                                transition: {
                                                    duration: 1,
                                                    delay: 1
                                                },
                                                children: [
                                                    [
                                                        ...Array(24)
                                                    ].map((_, i)=>{
                                                        const gearRadius = 145;
                                                        const toothHeight = 8;
                                                        const toothWidth = 12;
                                                        const angle = i * 360 / 24;
                                                        const angleRad = angle * Math.PI / 180;
                                                        const innerR = gearRadius - toothHeight / 2;
                                                        const outerR = gearRadius + toothHeight / 2;
                                                        // Each tooth is a small rectangle extending outward
                                                        const x1 = Math.cos(angleRad - 0.08) * innerR;
                                                        const y1 = Math.sin(angleRad - 0.08) * innerR;
                                                        const x2 = Math.cos(angleRad + 0.08) * innerR;
                                                        const y2 = Math.sin(angleRad + 0.08) * innerR;
                                                        const x3 = Math.cos(angleRad + 0.06) * outerR;
                                                        const y3 = Math.sin(angleRad + 0.06) * outerR;
                                                        const x4 = Math.cos(angleRad - 0.06) * outerR;
                                                        const y4 = Math.sin(angleRad - 0.06) * outerR;
                                                        return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("polygon", {
                                                            points: `${x1},${y1} ${x2},${y2} ${x3},${y3} ${x4},${y4}`,
                                                            fill: "rgba(0, 200, 255, 0.25)",
                                                            stroke: "rgba(0, 200, 255, 0.4)",
                                                            strokeWidth: "0.5"
                                                        }, `tooth-${i}`, false, {
                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                            lineNumber: 241,
                                                            columnNumber: 17
                                                        }, this);
                                                    }),
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("circle", {
                                                        cx: 0,
                                                        cy: 0,
                                                        r: 137,
                                                        fill: "none",
                                                        stroke: "rgba(0, 200, 255, 0.3)",
                                                        strokeWidth: "1"
                                                    }, void 0, false, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 251,
                                                        columnNumber: 13
                                                    }, this),
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("circle", {
                                                        cx: 0,
                                                        cy: 0,
                                                        r: 153,
                                                        fill: "none",
                                                        stroke: "rgba(0, 200, 255, 0.3)",
                                                        strokeWidth: "1"
                                                    }, void 0, false, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 260,
                                                        columnNumber: 13
                                                    }, this)
                                                ]
                                            }, void 0, true, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 215,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("defs", {
                                                children: [
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("filter", {
                                                        id: "glow",
                                                        x: "-50%",
                                                        y: "-50%",
                                                        width: "200%",
                                                        height: "200%",
                                                        children: [
                                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feGaussianBlur", {
                                                                stdDeviation: "3",
                                                                result: "coloredBlur"
                                                            }, void 0, false, {
                                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                lineNumber: 273,
                                                                columnNumber: 15
                                                            }, this),
                                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feMerge", {
                                                                children: [
                                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feMergeNode", {
                                                                        in: "coloredBlur"
                                                                    }, void 0, false, {
                                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                        lineNumber: 275,
                                                                        columnNumber: 17
                                                                    }, this),
                                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feMergeNode", {
                                                                        in: "SourceGraphic"
                                                                    }, void 0, false, {
                                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                        lineNumber: 276,
                                                                        columnNumber: 17
                                                                    }, this)
                                                                ]
                                                            }, void 0, true, {
                                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                lineNumber: 274,
                                                                columnNumber: 15
                                                            }, this)
                                                        ]
                                                    }, void 0, true, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 272,
                                                        columnNumber: 13
                                                    }, this),
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("filter", {
                                                        id: "strongGlow",
                                                        x: "-50%",
                                                        y: "-50%",
                                                        width: "200%",
                                                        height: "200%",
                                                        children: [
                                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feGaussianBlur", {
                                                                stdDeviation: "6",
                                                                result: "coloredBlur"
                                                            }, void 0, false, {
                                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                lineNumber: 280,
                                                                columnNumber: 15
                                                            }, this),
                                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feMerge", {
                                                                children: [
                                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feMergeNode", {
                                                                        in: "coloredBlur"
                                                                    }, void 0, false, {
                                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                        lineNumber: 282,
                                                                        columnNumber: 17
                                                                    }, this),
                                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("feMergeNode", {
                                                                        in: "SourceGraphic"
                                                                    }, void 0, false, {
                                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                        lineNumber: 283,
                                                                        columnNumber: 17
                                                                    }, this)
                                                                ]
                                                            }, void 0, true, {
                                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                lineNumber: 281,
                                                                columnNumber: 15
                                                            }, this)
                                                        ]
                                                    }, void 0, true, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 279,
                                                        columnNumber: 13
                                                    }, this)
                                                ]
                                            }, void 0, true, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 271,
                                                columnNumber: 11
                                            }, this),
                                            [
                                                ...Array(16)
                                            ].map((_, i)=>{
                                                const angle = i * 360 / 16;
                                                const pos1 = polarToCartesian(angle, ring1Radius);
                                                const pos2 = polarToCartesian(angle, ring2Radius);
                                                const pos3 = polarToCartesian(angle, ring3Radius);
                                                const pos4 = polarToCartesian(angle, ring4Radius);
                                                // Cardinal points align with icons (every other dot: 0, 2, 4, 6, 8, 10, 12, 14)
                                                const isCardinal = i % 2 === 0;
                                                return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("g", {
                                                    children: [
                                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                            cx: pos1.x,
                                                            cy: pos1.y,
                                                            r: isCardinal ? 4.5 : 3,
                                                            fill: "#00d4ff",
                                                            filter: "url(#glow)",
                                                            initial: {
                                                                scale: 0,
                                                                opacity: 0
                                                            },
                                                            animate: mounted ? {
                                                                scale: 1,
                                                                opacity: isCardinal ? 1 : 0.8
                                                            } : {},
                                                            transition: {
                                                                duration: 0.3,
                                                                delay: 1 + i * 0.05
                                                            }
                                                        }, void 0, false, {
                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                            lineNumber: 299,
                                                            columnNumber: 17
                                                        }, this),
                                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                            cx: pos2.x,
                                                            cy: pos2.y,
                                                            r: 2.5,
                                                            fill: "#00d4ff",
                                                            filter: "url(#glow)",
                                                            initial: {
                                                                scale: 0,
                                                                opacity: 0
                                                            },
                                                            animate: mounted ? {
                                                                scale: 1,
                                                                opacity: 0.6
                                                            } : {},
                                                            transition: {
                                                                duration: 0.3,
                                                                delay: 1.2 + i * 0.05
                                                            }
                                                        }, void 0, false, {
                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                            lineNumber: 309,
                                                            columnNumber: 17
                                                        }, this),
                                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                            cx: pos3.x,
                                                            cy: pos3.y,
                                                            r: 2,
                                                            fill: "#00d4ff",
                                                            filter: "url(#glow)",
                                                            initial: {
                                                                scale: 0,
                                                                opacity: 0
                                                            },
                                                            animate: mounted ? {
                                                                scale: 1,
                                                                opacity: 0.5
                                                            } : {},
                                                            transition: {
                                                                duration: 0.3,
                                                                delay: 1.4 + i * 0.05
                                                            }
                                                        }, void 0, false, {
                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                            lineNumber: 319,
                                                            columnNumber: 17
                                                        }, this),
                                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                            cx: pos4.x,
                                                            cy: pos4.y,
                                                            r: 1.5,
                                                            fill: "#00d4ff",
                                                            initial: {
                                                                scale: 0,
                                                                opacity: 0
                                                            },
                                                            animate: mounted ? {
                                                                scale: 1,
                                                                opacity: 0.4
                                                            } : {},
                                                            transition: {
                                                                duration: 0.3,
                                                                delay: 1.6 + i * 0.05
                                                            }
                                                        }, void 0, false, {
                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                            lineNumber: 329,
                                                            columnNumber: 17
                                                        }, this)
                                                    ]
                                                }, `nodes-${i}`, true, {
                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                    lineNumber: 298,
                                                    columnNumber: 15
                                                }, this);
                                            }),
                                            [
                                                ...Array(12)
                                            ].map((_, i)=>{
                                                const angle = i * 360 / 12;
                                                const inner = polarToCartesian(angle, ring1Radius - 20);
                                                const outer = polarToCartesian(angle, ring4Radius + 20);
                                                return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].line, {
                                                    x1: inner.x,
                                                    y1: inner.y,
                                                    x2: outer.x,
                                                    y2: outer.y,
                                                    stroke: "rgba(0, 200, 255, 0.1)",
                                                    strokeWidth: "1",
                                                    initial: {
                                                        pathLength: 0,
                                                        opacity: 0
                                                    },
                                                    animate: loadingComplete ? {
                                                        pathLength: 1,
                                                        opacity: 1
                                                    } : {},
                                                    transition: {
                                                        duration: 1,
                                                        delay: 1.5 + i * 0.05
                                                    }
                                                }, `radial-${i}`, false, {
                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                    lineNumber: 348,
                                                    columnNumber: 15
                                                }, this);
                                            }),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].g, {
                                                animate: loadingComplete ? {
                                                    rotate: 360
                                                } : {},
                                                transition: {
                                                    duration: 120,
                                                    repeat: Number.POSITIVE_INFINITY,
                                                    ease: "linear"
                                                },
                                                children: [
                                                    45,
                                                    135,
                                                    225,
                                                    315
                                                ].map((startAngle, i)=>{
                                                    const r = ring3Radius + 15;
                                                    const start = polarToCartesian(startAngle, r);
                                                    const end = polarToCartesian(startAngle + 30, r);
                                                    return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].path, {
                                                        d: `M ${start.x} ${start.y} A ${r} ${r} 0 0 1 ${end.x} ${end.y}`,
                                                        fill: "none",
                                                        stroke: "rgba(0, 200, 255, 0.4)",
                                                        strokeWidth: "3",
                                                        filter: "url(#glow)",
                                                        initial: {
                                                            pathLength: 0,
                                                            opacity: 0
                                                        },
                                                        animate: loadingComplete ? {
                                                            pathLength: 1,
                                                            opacity: 1
                                                        } : {},
                                                        transition: {
                                                            duration: 0.8,
                                                            delay: 2 + i * 0.1
                                                        }
                                                    }, `arc-${i}`, false, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 377,
                                                        columnNumber: 17
                                                    }, this);
                                                })
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 364,
                                                columnNumber: 11
                                            }, this),
                                            mounted && /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["Fragment"], {
                                                children: [
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                        cx: 0,
                                                        cy: 0,
                                                        r: ring2Radius,
                                                        fill: "none",
                                                        stroke: "rgba(0, 240, 255, 0.6)",
                                                        strokeWidth: "2",
                                                        strokeDasharray: "10 20",
                                                        filter: "url(#strongGlow)",
                                                        animate: {
                                                            rotate: [
                                                                0,
                                                                360
                                                            ]
                                                        },
                                                        transition: {
                                                            duration: 20,
                                                            repeat: Infinity,
                                                            ease: "linear"
                                                        },
                                                        style: {
                                                            transformOrigin: "center"
                                                        }
                                                    }, void 0, false, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 395,
                                                        columnNumber: 15
                                                    }, this),
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].circle, {
                                                        cx: 0,
                                                        cy: 0,
                                                        r: ring3Radius,
                                                        fill: "none",
                                                        stroke: "rgba(0, 240, 255, 0.4)",
                                                        strokeWidth: "1.5",
                                                        strokeDasharray: "5 30",
                                                        filter: "url(#glow)",
                                                        animate: {
                                                            rotate: [
                                                                360,
                                                                0
                                                            ]
                                                        },
                                                        transition: {
                                                            duration: 30,
                                                            repeat: Infinity,
                                                            ease: "linear"
                                                        },
                                                        style: {
                                                            transformOrigin: "center"
                                                        }
                                                    }, void 0, false, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 414,
                                                        columnNumber: 15
                                                    }, this)
                                                ]
                                            }, void 0, true)
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 130,
                                        columnNumber: 9
                                    }, this),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                        className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",
                                        style: {
                                            opacity: loadingComplete ? 1 : 0,
                                            transition: 'opacity 1.5s ease-in-out'
                                        },
                                        children: [
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                                className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",
                                                style: {
                                                    width: centerSize + 16,
                                                    height: centerSize + 16,
                                                    background: "radial-gradient(circle, transparent 65%, rgba(0, 220, 255, 0.5) 85%, rgba(0, 220, 255, 0.8) 95%, rgba(0, 200, 255, 0.4) 100%)",
                                                    filter: "blur(4px)"
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 446,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                                className: "relative flex items-center justify-center rounded-full",
                                                style: {
                                                    width: centerSize,
                                                    height: centerSize,
                                                    background: "#030a18",
                                                    border: "2px solid rgba(0, 220, 255, 0.8)",
                                                    boxShadow: "0 0 15px rgba(0, 220, 255, 0.6), 0 0 30px rgba(0, 220, 255, 0.3)"
                                                },
                                                children: [
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("svg", {
                                                        className: "absolute overflow-visible",
                                                        width: centerSize + 20,
                                                        height: centerSize + 20,
                                                        viewBox: "-10 -10 180 180",
                                                        style: {
                                                            left: -10,
                                                            top: -10
                                                        },
                                                        children: [
                                                            [
                                                                ...Array(90)
                                                            ].map((_, i)=>{
                                                                const seed = i * 7919 + 1234;
                                                                const x = 15 + (seed * 13 + i * 23) % 130;
                                                                const y = 15 + (seed * 17 + i * 31) % 130;
                                                                // Smaller dots, slightly larger near edges
                                                                const distFromCenter = Math.sqrt(Math.pow(x - 80, 2) + Math.pow(y - 80, 2));
                                                                const size = distFromCenter > 60 ? 0.8 + i % 3 * 0.3 : 0.5 + i % 3 * 0.2;
                                                                return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("circle", {
                                                                    cx: x,
                                                                    cy: y,
                                                                    r: size,
                                                                    fill: distFromCenter > 65 ? "rgba(0, 220, 255, 0.8)" : "rgba(0, 220, 255, 0.5)"
                                                                }, `node-${i}`, false, {
                                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                    lineNumber: 484,
                                                                    columnNumber: 19
                                                                }, this);
                                                            }),
                                                            [
                                                                ...Array(60)
                                                            ].map((_, i)=>{
                                                                const angle = (i * 6 + i * 7919 % 4) * (Math.PI / 180);
                                                                const radius = 78 + i * 7919 % 12;
                                                                const x = 80 + Math.cos(angle) * radius;
                                                                const y = 80 + Math.sin(angle) * radius;
                                                                return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("circle", {
                                                                    cx: x,
                                                                    cy: y,
                                                                    r: 0.6 + i % 2 * 0.3,
                                                                    fill: "rgba(0, 220, 255, 0.7)"
                                                                }, `edge-${i}`, false, {
                                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                    lineNumber: 501,
                                                                    columnNumber: 19
                                                                }, this);
                                                            }),
                                                            [
                                                                ...Array(100)
                                                            ].map((_, i)=>{
                                                                const seed1 = i * 7919;
                                                                const seed2 = (i + 3) % 100 * 7919;
                                                                const x1 = 15 + (seed1 * 13 + i * 17) % 130;
                                                                const y1 = 15 + (seed1 * 17 + i * 23) % 130;
                                                                const x2 = 15 + (seed2 * 13 + i * 29) % 130;
                                                                const y2 = 15 + (seed2 * 17 + i * 31) % 130;
                                                                return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("line", {
                                                                    x1: x1,
                                                                    y1: y1,
                                                                    x2: x2,
                                                                    y2: y2,
                                                                    stroke: "rgba(0, 200, 255, 0.3)",
                                                                    strokeWidth: "0.4"
                                                                }, `mesh-${i}`, false, {
                                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                    lineNumber: 520,
                                                                    columnNumber: 19
                                                                }, this);
                                                            }),
                                                            [
                                                                ...Array(50)
                                                            ].map((_, i)=>{
                                                                const seed1 = i * 2 * 7919;
                                                                const seed2 = (i * 2 + 5) % 90 * 7919;
                                                                const seed3 = (i * 2 + 11) % 90 * 7919;
                                                                const x1 = 20 + seed1 * 11 % 120;
                                                                const y1 = 20 + seed1 * 19 % 120;
                                                                const x2 = 20 + seed2 * 11 % 120;
                                                                const y2 = 20 + seed2 * 19 % 120;
                                                                const x3 = 20 + seed3 * 11 % 120;
                                                                const y3 = 20 + seed3 * 19 % 120;
                                                                return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("g", {
                                                                    children: [
                                                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("line", {
                                                                            x1: x1,
                                                                            y1: y1,
                                                                            x2: x2,
                                                                            y2: y2,
                                                                            stroke: "rgba(0, 200, 255, 0.25)",
                                                                            strokeWidth: "0.35"
                                                                        }, void 0, false, {
                                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                            lineNumber: 545,
                                                                            columnNumber: 21
                                                                        }, this),
                                                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("line", {
                                                                            x1: x2,
                                                                            y1: y2,
                                                                            x2: x3,
                                                                            y2: y3,
                                                                            stroke: "rgba(0, 200, 255, 0.25)",
                                                                            strokeWidth: "0.35"
                                                                        }, void 0, false, {
                                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                            lineNumber: 546,
                                                                            columnNumber: 21
                                                                        }, this),
                                                                        /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("line", {
                                                                            x1: x3,
                                                                            y1: y3,
                                                                            x2: x1,
                                                                            y2: y1,
                                                                            stroke: "rgba(0, 200, 255, 0.25)",
                                                                            strokeWidth: "0.35"
                                                                        }, void 0, false, {
                                                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                            lineNumber: 547,
                                                                            columnNumber: 21
                                                                        }, this)
                                                                    ]
                                                                }, `tri-${i}`, true, {
                                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                    lineNumber: 544,
                                                                    columnNumber: 19
                                                                }, this);
                                                            }),
                                                            [
                                                                ...Array(6)
                                                            ].map((_, i)=>{
                                                                const seed1 = i * 7 * 7919;
                                                                const seed2 = (i * 7 + 4) * 7919;
                                                                const x1 = 25 + seed1 * 11 % 110;
                                                                const y1 = 25 + seed1 * 19 % 110;
                                                                const x2 = 25 + seed2 * 11 % 110;
                                                                const y2 = 25 + seed2 * 19 % 110;
                                                                return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("line", {
                                                                    x1: x1,
                                                                    y1: y1,
                                                                    x2: x2,
                                                                    y2: y2,
                                                                    stroke: "rgba(180, 100, 220, 0.45)",
                                                                    strokeWidth: "0.5"
                                                                }, `purple-${i}`, false, {
                                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                                    lineNumber: 561,
                                                                    columnNumber: 19
                                                                }, this);
                                                            })
                                                        ]
                                                    }, void 0, true, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 468,
                                                        columnNumber: 13
                                                    }, this),
                                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("span", {
                                                        className: "relative z-10 text-xl font-bold tracking-[0.15em] text-white",
                                                        style: {
                                                            textShadow: "0 0 10px rgba(255, 255, 255, 0.8), 0 0 20px rgba(0, 200, 255, 0.6)"
                                                        },
                                                        children: "KNIRVANA"
                                                    }, void 0, false, {
                                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                        lineNumber: 575,
                                                        columnNumber: 13
                                                    }, this)
                                                ]
                                            }, void 0, true, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 457,
                                                columnNumber: 11
                                            }, this),
                                            /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])("div", {
                                                className: "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",
                                                style: {
                                                    width: centerSize + 24,
                                                    height: centerSize + 24,
                                                    border: "2px solid rgba(0, 220, 255, 0.6)",
                                                    boxShadow: "0 0 30px rgba(0, 220, 255, 0.6), 0 0 60px rgba(0, 200, 255, 0.4), inset 0 0 20px rgba(0, 220, 255, 0.3)"
                                                }
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 586,
                                                columnNumber: 11
                                            }, this)
                                        ]
                                    }, void 0, true, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 438,
                                        columnNumber: 10
                                    }, this),
                                    innerLabels.map((item, i)=>{
                                        const pos = polarToCartesian(item.angle, ring1Radius + 35);
                                        return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                                            className: "absolute left-1/2 top-1/2 whitespace-nowrap text-[9px] font-medium tracking-wider",
                                            style: {
                                                transform: `translate(calc(-50% + ${pos.x}px), calc(-50% + ${pos.y}px))`,
                                                color: "rgba(0, 220, 255, 0.9)",
                                                textShadow: "0 0 8px rgba(0, 200, 255, 0.6)"
                                            },
                                            initial: {
                                                opacity: 0
                                            },
                                            animate: loadingComplete ? {
                                                opacity: 1
                                            } : {},
                                            transition: {
                                                duration: 0.5,
                                                delay: 2 + i * 0.05
                                            },
                                            children: item.label
                                        }, `inner-${i}`, false, {
                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                            lineNumber: 601,
                                            columnNumber: 13
                                        }, this);
                                    }),
                                    middleLabels.map((item, i)=>{
                                        const pos = polarToCartesian(item.angle, ring2Radius + 40);
                                        return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                                            className: "absolute left-1/2 top-1/2 whitespace-nowrap text-[10px] font-semibold tracking-wider",
                                            style: {
                                                transform: `translate(calc(-50% + ${pos.x}px), calc(-50% + ${pos.y}px))`,
                                                color: "rgba(0, 230, 255, 1)",
                                                textShadow: "0 0 10px rgba(0, 200, 255, 0.8)"
                                            },
                                            initial: {
                                                opacity: 0
                                            },
                                            animate: loadingComplete ? {
                                                opacity: 1
                                            } : {},
                                            transition: {
                                                duration: 0.5,
                                                delay: 2.3 + i * 0.05
                                            },
                                            children: item.label
                                        }, `middle-${i}`, false, {
                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                            lineNumber: 622,
                                            columnNumber: 13
                                        }, this);
                                    }),
                                    outerIcons.map((item, i)=>{
                                        const pos = polarToCartesian(item.angle, iconRadius);
                                        const IconComponent = item.icon;
                                        const isHovered = hoveredIcon === i;
                                        // Calculate absolute position from center (450 is half of 900)
                                        const absoluteX = 450 + pos.x - 28 // 28 is half of icon size (56)
                                        ;
                                        const absoluteY = 450 + pos.y - 28;
                                        const isSettingsIcon = item.icon === __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$lucide$2d$react$2f$dist$2f$esm$2f$icons$2f$settings$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__$3c$export__default__as__Settings$3e$__["Settings"];
                                        return /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                                            className: "absolute cursor-pointer",
                                            style: {
                                                left: absoluteX,
                                                top: absoluteY
                                            },
                                            initial: {
                                                opacity: 0,
                                                scale: 0
                                            },
                                            animate: loadingComplete ? {
                                                opacity: 1,
                                                scale: 1
                                            } : {},
                                            transition: {
                                                duration: 0.5,
                                                delay: 2.6 + i * 0.1,
                                                type: "spring"
                                            },
                                            onMouseEnter: ()=>setHoveredIcon(i),
                                            onMouseLeave: ()=>setHoveredIcon(null),
                                            onClick: ()=>{
                                                if (isSettingsIcon) {
                                                    setSettingsOpen(true);
                                                }
                                            },
                                            children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                                                className: "flex items-center justify-center rounded-full",
                                                style: {
                                                    width: 56,
                                                    height: 56,
                                                    background: isHovered ? "rgba(0, 100, 150, 0.6)" : "rgba(0, 50, 80, 0.4)",
                                                    backdropFilter: "blur(8px)",
                                                    border: `2px solid ${isHovered ? "rgba(0, 220, 255, 0.8)" : "rgba(0, 200, 255, 0.4)"}`,
                                                    boxShadow: isHovered ? "0 0 30px rgba(0, 200, 255, 0.7), 0 0 60px rgba(0, 200, 255, 0.4), inset 0 0 20px rgba(0, 200, 255, 0.3)" : "0 0 15px rgba(0, 200, 255, 0.3), inset 0 0 10px rgba(0, 200, 255, 0.1)"
                                                },
                                                animate: {
                                                    scale: isHovered ? 1.15 : 1
                                                },
                                                transition: {
                                                    duration: 0.2
                                                },
                                                children: /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(IconComponent, {
                                                    className: "h-6 w-6",
                                                    style: {
                                                        color: isHovered ? "#00f0ff" : "rgba(0, 220, 255, 0.8)",
                                                        filter: isHovered ? "drop-shadow(0 0 10px rgba(0, 240, 255, 1))" : "drop-shadow(0 0 4px rgba(0, 200, 255, 0.5))"
                                                    }
                                                }, void 0, false, {
                                                    fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                    lineNumber: 687,
                                                    columnNumber: 17
                                                }, this)
                                            }, void 0, false, {
                                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                                lineNumber: 668,
                                                columnNumber: 15
                                            }, this)
                                        }, `icon-${i}`, false, {
                                            fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                            lineNumber: 650,
                                            columnNumber: 13
                                        }, this);
                                    }),
                                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$framer$2d$motion$2f$dist$2f$es$2f$render$2f$components$2f$motion$2f$proxy$2e$mjs__$5b$app$2d$client$5d$__$28$ecmascript$29$__["motion"].div, {
                                        className: "absolute bottom-4 left-1/2 -translate-x-1/2 text-2xl font-bold tracking-[0.4em]",
                                        style: {
                                            color: "rgba(255, 255, 255, 0.95)",
                                            textShadow: "0 0 20px rgba(0, 200, 255, 0.6), 0 0 40px rgba(0, 200, 255, 0.3)"
                                        },
                                        initial: {
                                            opacity: 0,
                                            y: 20
                                        },
                                        animate: loadingComplete ? {
                                            opacity: 1,
                                            y: 0
                                        } : {},
                                        transition: {
                                            duration: 0.8,
                                            delay: 3
                                        },
                                        children: "KNIRV.COM"
                                    }, void 0, false, {
                                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                        lineNumber: 702,
                                        columnNumber: 9
                                    }, this)
                                ]
                            }, void 0, true, {
                                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                                lineNumber: 125,
                                columnNumber: 7
                            }, this)
                        ]
                    }, void 0, true, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                        lineNumber: 86,
                        columnNumber: 9
                    }, this),
                    /*#__PURE__*/ (0, __TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$node_modules$2f$next$2f$dist$2f$compiled$2f$react$2f$jsx$2d$dev$2d$runtime$2e$js__$5b$app$2d$client$5d$__$28$ecmascript$29$__["jsxDEV"])(__TURBOPACK__imported__module__$5b$project$5d2f$Documents$2f$GitHub$2f$KNIRV$2f$KNIRV_NETWORK$2f$packages$2f$KNIRVSERVER$2f$desktop$2f$menu$2f$components$2f$settings$2d$modal$2e$tsx__$5b$app$2d$client$5d$__$28$ecmascript$29$__["default"], {
                        isOpen: settingsOpen,
                        onClose: ()=>setSettingsOpen(false)
                    }, void 0, false, {
                        fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                        lineNumber: 718,
                        columnNumber: 7
                    }, this)
                ]
            }, void 0, true, {
                fileName: "[project]/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/desktop/menu/components/constellation-menu.tsx",
                lineNumber: 82,
                columnNumber: 7
            }, this)
        ]
    }, void 0, true);
}
_s(ConstellationMenu, "MZySv/YTv5palHOogEZ/4H22j3M=");
_c = ConstellationMenu;
var _c;
__turbopack_context__.k.register(_c, "ConstellationMenu");
if (typeof globalThis.$RefreshHelpers$ === 'object' && globalThis.$RefreshHelpers !== null) {
    __turbopack_context__.k.registerExports(__turbopack_context__.m, globalThis.$RefreshHelpers$);
}
}),
]);

//# sourceMappingURL=%5Broot-of-the-server%5D__f161baf1._.js.map