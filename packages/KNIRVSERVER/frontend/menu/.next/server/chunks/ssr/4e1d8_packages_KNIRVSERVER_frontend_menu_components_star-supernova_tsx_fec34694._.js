module.exports=[57985,a=>a.a(async(b,c)=>{try{var d=a.i(8171),e=a.i(27669),f=a.i(34791),g=b([f]);function h({onComplete:a}){let[b,c]=(0,e.useState)("pulsing"),[g,h]=(0,e.useState)([]),[i,j]=(0,e.useState)(0);return((0,e.useEffect)(()=>{let b=setTimeout(()=>{c("supernova");let b=[];for(let a=0;a<200;a++){let c=2*Math.PI*a/200+(Math.random()-.5)*.5,d=1.5+3*Math.random(),e=2.5*Math.random()+.5,f=Math.random(),g="rgba(72, 136, 255, ";f>.7?g="rgba(100, 150, 255, ":f>.4&&(g="rgba(72, 136, 255, ");let h=2*Math.PI*a/200,i=150+200*Math.random(),j=Math.cos(h)*i,k=Math.sin(h)*i;b.push({id:a,x:0,y:0,vx:Math.cos(c)*d,vy:Math.sin(c)*d,size:e,opacity:1,color:g,targetX:j,targetY:k})}h(b),setTimeout(()=>{c("forming"),j(1),setTimeout(a,1500)},800)},2e3);return()=>clearTimeout(b)},[a]),"complete"===b)?null:(0,d.jsxs)("div",{className:"fixed inset-0 z-50 flex items-center justify-center overflow-hidden",children:[(0,d.jsx)("div",{className:"absolute inset-0",style:{backgroundColor:"#030a18"},children:[...Array(100)].map((a,b)=>(0,d.jsx)(f.motion.div,{className:"absolute rounded-full bg-white",style:{left:`${100*Math.random()}%`,top:`${100*Math.random()}%`,width:Math.random()>.8?2:1,height:Math.random()>.8?2:1,opacity:.4*Math.random()+.1},animate:{opacity:[.1,.4,.1]},transition:{duration:3*Math.random()+2,repeat:1/0,ease:"easeInOut",delay:2*Math.random()}},b))}),(0,d.jsxs)(f.motion.div,{className:"relative",style:{width:100,height:100},animate:"pulsing"===b?{scale:[1,1.2,1]}:"supernova"===b?{scale:[1,3,0],opacity:[1,.8,0]}:{},transition:"pulsing"===b?{duration:2,repeat:1/0,ease:"easeInOut"}:{duration:1,ease:"easeOut"},children:[(0,d.jsx)("div",{className:"absolute inset-0 rounded-full",style:{background:`
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
            `}}),(0,d.jsx)(f.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",style:{width:40,height:40,background:`
              radial-gradient(circle,
                rgba(72, 136, 255, 1) 0%,
                rgba(72, 136, 255, 0.9) 30%,
                rgba(72, 136, 255, 0.6) 60%,
                transparent 100%
              )
            `,filter:"blur(2px)"},animate:"pulsing"===b?{scale:[1,1.3,1],opacity:[.9,1,.9]}:{},transition:{duration:2,repeat:1/0,ease:"easeInOut"}}),(0,d.jsx)(f.motion.div,{className:"absolute inset-0 rounded-full",style:{background:`
              radial-gradient(circle,
                transparent 40%,
                rgba(72, 136, 255, 0.1) 60%,
                rgba(72, 136, 255, 0.05) 80%,
                transparent 100%
              )
            `,filter:"blur(8px)"},animate:"pulsing"===b?{scale:[1,1.4,1],opacity:[.5,.8,.5]}:{},transition:{duration:2.5,repeat:1/0,ease:"easeInOut"}})]}),"pulsing"===b&&(0,d.jsxs)(d.Fragment,{children:[(0,d.jsx)(f.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border-2",style:{width:140,height:140,borderColor:"rgba(72, 136, 255, 0.3)",boxShadow:"0 0 20px rgba(72, 136, 255, 0.2)"},animate:{scale:[1,1.6,1],opacity:[.3,.1,.3]},transition:{duration:2,repeat:1/0,ease:"easeInOut"}}),(0,d.jsx)(f.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border",style:{width:180,height:180,borderColor:"rgba(72, 136, 255, 0.2)",boxShadow:"0 0 30px rgba(72, 136, 255, 0.1)"},animate:{scale:[1,1.4,1],opacity:[.2,.05,.2]},transition:{duration:2.5,repeat:1/0,ease:"easeInOut"}})]}),"pulsing"!==b&&g.map(a=>(0,d.jsx)(f.motion.div,{className:"absolute rounded-full",style:{width:a.size,height:a.size,backgroundColor:a.color+"1)",boxShadow:`0 0 ${3*a.size}px ${a.color+"0.8)"}`},initial:{x:0,y:0,opacity:1},animate:"supernova"===b?{x:150*a.vx,y:150*a.vy,opacity:.8,scale:[1,1.2,1]}:{x:a.targetX||0,y:a.targetY||0,opacity:.3,scale:[1,.8,.6]},transition:"supernova"===b?{duration:.8,ease:"easeOut"}:{duration:1.5,ease:"easeInOut"}},a.id)),"forming"===b&&(0,d.jsxs)(f.motion.div,{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",style:{width:900,height:900},animate:{scale:i,opacity:.3*i},transition:{duration:1.5,ease:"easeOut"},children:[[120,180,250,320].map((a,b)=>(0,d.jsx)("div",{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border",style:{width:2*a,height:2*a,borderColor:"rgba(72, 136, 255, 0.2)",boxShadow:`0 0 ${20-3*b}px rgba(72, 136, 255, 0.1)`}},b)),(0,d.jsx)("div",{className:"absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full",style:{width:160,height:160,background:"rgba(0, 100, 150, 0.1)",border:"2px solid rgba(72, 136, 255, 0.3)",boxShadow:"0 0 30px rgba(72, 136, 255, 0.2)"}})]})]})}[f]=g.then?(await g)():g,a.s(["default",()=>h]),c()}catch(a){c(a)}},!1),71686,a=>{a.n(a.i(57985))}];

//# sourceMappingURL=4e1d8_packages_KNIRVSERVER_frontend_menu_components_star-supernova_tsx_fec34694._.js.map