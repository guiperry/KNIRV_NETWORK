(globalThis.TURBOPACK||(globalThis.TURBOPACK=[])).push(["object"==typeof document?document.currentScript:void 0,92360,a=>{"use strict";var t=a.i(3487),e=a.i(89679),r=a.i(79133);function s({onComplete:a}){let[s,o]=(0,e.useState)("pulsing"),[i,n]=(0,e.useState)([]),[l,d]=(0,e.useState)(0);return((0,e.useEffect)(()=>{let t=setTimeout(()=>{o("supernova");let t=[];for(let a=0;a<200;a++){let e=2*Math.PI*a/200+(Math.random()-.5)*.5,r=1.5+3*Math.random(),s=2.5*Math.random()+.5,o=Math.random(),i="rgba(72, 136, 255, ";o>.7?i="rgba(100, 150, 255, ":o>.4&&(i="rgba(72, 136, 255, ");let n=2*Math.PI*a/200,l=150+200*Math.random(),d=Math.cos(n)*l,u=Math.sin(n)*l;t.push({id:a,x:0,y:0,vx:Math.cos(e)*r,vy:Math.sin(e)*r,size:s,opacity:1,color:i,targetX:d,targetY:u})}n(t),setTimeout(()=>{o("forming"),d(1),setTimeout(a,1500)},800)},2e3);return()=>clearTimeout(t)},[a]),"complete"===s)?null:(0,t.jsxs)("div",{className:"fixed inset-0 z-50 flex items-center justify-center overflow-hidden",children:[(0,t.jsx)("div",{className:"absolute inset-0",style:{backgroundColor:"#030a18"},children:[...Array(100)].map((a,e)=>(0,t.jsx)(r.motion.div,{className:"absolute rounded-full bg-white",style:{left:`${100*Math.random()}%`,top:`${100*Math.random()}%`,width:Math.random()>.8?2:1,height:Math.random()>.8?2:1,opacity:.4*Math.random()+.1},animate:{opacity:[.1,.4,.1]},transition:{duration:3*Math.random()+2,repeat:1/0,ease:"easeInOut",delay:2*Math.random()}},e))}),(0,t.jsxs)(r.motion.div,{className:"relative",style:{width:100,height:100},animate:"pulsing"===s?{scale:[1,1.2,1]}:"supernova"===s?{scale:[1,3,0],opacity:[1,.8,0]}:{},transition:"pulsing"===s?{duration:2,repeat:1/0,ease:"easeInOut"}:{duration:1,ease:"easeOut"},children:[(0,t.jsx)("div",{className:"absolute inset-0 rounded-full",style:{background:`
              radial-gradient(circle at 30% 30%, 
                rgba(72, 136, 255, 0.9) 0%,
                rgba(72, 136, 255, 0.8) 10%,
                rgba(0, 180, 255, 0.7) 25%,
                rgba(72, 136, 255, 0.6) 50%,
                rgba(0, 100, 200, 0.4) 75%,
                transparent 100%
              )
            `,boxShadow:`
              0 0 60px rgba(72, 136, 255, 0.8),
              0 0 100px rgba(72, 136, 255, 0.6),
              0 0 150px rgba(0, 100, 200, 0.4),
              inset -10px -10px 20px rgba(0, 50, 150, 0.3)
            `}}),(0,t.jsx)(r.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",style:{width:40,height:40,background:`
              radial-gradient(circle,
                rgba(72, 136, 255, 1) 0%,
                rgba(72, 136, 255, 0.9) 30%,
                rgba(72, 136, 255, 0.6) 60%,
                transparent 100%
              )
            `,filter:"blur(2px)"},animate:"pulsing"===s?{scale:[1,1.3,1],opacity:[.9,1,.9]}:{},transition:{duration:2,repeat:1/0,ease:"easeInOut"}}),(0,t.jsx)(r.motion.div,{className:"absolute inset-0 rounded-full",style:{background:`
              radial-gradient(circle,
                transparent 40%,
                rgba(72, 136, 255, 0.1) 60%,
                rgba(72, 136, 255, 0.05) 80%,
                transparent 100%
              )
            `,filter:"blur(8px)"},animate:"pulsing"===s?{scale:[1,1.4,1],opacity:[.5,.8,.5]}:{},transition:{duration:2.5,repeat:1/0,ease:"easeInOut"}})]}),"pulsing"===s&&(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(r.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border-2",style:{width:140,height:140,borderColor:"rgba(72, 136, 255, 0.3)",boxShadow:"0 0 20px rgba(72, 136, 255, 0.2)"},animate:{scale:[1,1.6,1],opacity:[.3,.1,.3]},transition:{duration:2,repeat:1/0,ease:"easeInOut"}}),(0,t.jsx)(r.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border",style:{width:180,height:180,borderColor:"rgba(72, 136, 255, 0.2)",boxShadow:"0 0 30px rgba(72, 136, 255, 0.1)"},animate:{scale:[1,1.4,1],opacity:[.2,.05,.2]},transition:{duration:2.5,repeat:1/0,ease:"easeInOut"}})]}),"pulsing"!==s&&i.map(a=>(0,t.jsx)(r.motion.div,{className:"absolute rounded-full",style:{width:a.size,height:a.size,backgroundColor:a.color+"1)",boxShadow:`0 0 ${3*a.size}px ${a.color+"0.8)"}`},initial:{x:0,y:0,opacity:1},animate:"supernova"===s?{x:150*a.vx,y:150*a.vy,opacity:.8,scale:[1,1.2,1]}:{x:a.targetX||0,y:a.targetY||0,opacity:.3,scale:[1,.8,.6]},transition:"supernova"===s?{duration:.8,ease:"easeOut"}:{duration:1.5,ease:"easeInOut"}},a.id)),"forming"===s&&(0,t.jsxs)(r.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",style:{width:900,height:900},animate:{scale:l,opacity:.3*l},transition:{duration:1.5,ease:"easeOut"},children:[[120,180,250,320].map((a,e)=>(0,t.jsx)("div",{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border",style:{width:2*a,height:2*a,borderColor:"rgba(72, 136, 255, 0.2)",boxShadow:`0 0 ${20-3*e}px rgba(72, 136, 255, 0.1)`}},e)),(0,t.jsx)("div",{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",style:{width:160,height:160,background:"rgba(0, 100, 150, 0.1)",border:"2px solid rgba(72, 136, 255, 0.3)",boxShadow:"0 0 30px rgba(72, 136, 255, 0.2)"}})]})]})}a.s(["default",()=>s])},18159,a=>{a.n(a.i(92360))}]);