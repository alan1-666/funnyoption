(()=>{var e={};e.id=663,e.ids=[663],e.modules={846:e=>{"use strict";e.exports=require("next/dist/compiled/next-server/app-page.runtime.prod.js")},3033:e=>{"use strict";e.exports=require("next/dist/server/app-render/work-unit-async-storage.external.js")},3295:e=>{"use strict";e.exports=require("next/dist/server/app-render/after-task-async-storage.external.js")},3457:(e,t,r)=>{"use strict";r.r(t),r.d(t,{patchFetch:()=>h,routeModule:()=>d,serverHooks:()=>k,workAsyncStorage:()=>c,workUnitAsyncStorage:()=>m});var o={};r.r(o),r.d(o,{POST:()=>p});var s=r(4296),a=r(8093),n=r(8156),u=r(4257),i=r(6135),l=r(9034);async function p(e,{params:t}){let{marketId:r}=await t,o=Number.parseInt(r,10);if(!Number.isFinite(o)||o<=0)return u.NextResponse.json({error:"invalid marketId"},{status:400});let s=await e.json().catch(()=>null);if(!s?.market||!s?.operator)return u.NextResponse.json({error:"market and operator payloads are required"},{status:400});let a=(0,i.P4)({...s.market,marketId:o}),n=await (0,l.M0)((0,i.cD)({walletAddress:s.operator.walletAddress,market:a,requestedAt:s.operator.requestedAt}),s.operator);if(!n.ok)return n.response;let p=await (0,l.kR)(`/api/v1/markets/${a.marketId}/resolve`,{method:"POST",body:{outcome:a.outcome,operator:(0,l.VR)(s.operator)}});return p.ok?u.NextResponse.json({...p.payload,operator_wallet_address:n.walletAddress},{status:p.status}):p.response}let d=new s.AppRouteRouteModule({definition:{kind:a.RouteKind.APP_ROUTE,page:"/api/operator/markets/[marketId]/resolve/route",pathname:"/api/operator/markets/[marketId]/resolve",filename:"route",bundlePath:"app/api/operator/markets/[marketId]/resolve/route"},resolvedPagePath:"/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/[marketId]/resolve/route.ts",nextConfigOutput:"",userland:o}),{workAsyncStorage:c,workUnitAsyncStorage:m,serverHooks:k}=d;function h(){return(0,n.patchFetch)({workAsyncStorage:c,workUnitAsyncStorage:m})}},4870:e=>{"use strict";e.exports=require("next/dist/compiled/next-server/app-route.runtime.prod.js")},5620:()=>{},6135:(e,t,r)=>{"use strict";r.d(t,{P4:()=>i,Qh:()=>a,WZ:()=>p,Xt:()=>l,_X:()=>n,cD:()=>c,iH:()=>o,ph:()=>u,wy:()=>d});let o=3e5;function s(e){return e.trim().replace(/\s+/g," ")}function a(e){return e.trim().toLowerCase()}function n(e){return e.split(",").map(e=>a(e)).filter(Boolean)}function u(e){return{title:s(e.title),description:s(e.description),category:s(e.category)||"Polymarket",coverImage:e.coverImage.trim(),sourceUrl:e.sourceUrl.trim(),sourceSlug:s(e.sourceSlug),sourceName:s(e.sourceName)||"Polymarket",sourceKind:s(e.sourceKind).toLowerCase()||"manual",status:s(e.status).toUpperCase()||"OPEN",collateralAsset:s(e.collateralAsset).toUpperCase()||"USDT",openAt:Math.max(0,Math.floor(e.openAt||0)),closeAt:Math.max(0,Math.floor(e.closeAt||0)),resolveAt:Math.max(0,Math.floor(e.resolveAt||0))}}function i(e){return{marketId:Math.max(0,Math.floor(e.marketId||0)),outcome:"NO"===s(e.outcome).toUpperCase()?"NO":"YES"}}function l(e){return{marketId:Math.max(0,Math.floor(e.marketId||0)),userId:Math.max(0,Math.floor(e.userId||0)),quantity:Math.max(0,Math.floor(e.quantity||0)),outcome:"NO"===s(e.outcome).toUpperCase()?"NO":"YES",price:Math.max(0,Math.floor(e.price||0))}}function p(e){let t=a(e.walletAddress),r=u(e.market);return`FunnyOption Operator Authorization

action: CREATE_MARKET
wallet: ${t}
title: ${r.title}
description: ${r.description}
category: ${r.category}
source_kind: ${r.sourceKind}
source_url: ${r.sourceUrl}
source_slug: ${r.sourceSlug}
source_name: ${r.sourceName}
cover_image: ${r.coverImage}
status: ${r.status}
collateral_asset: ${r.collateralAsset}
open_at: ${r.openAt}
close_at: ${r.closeAt}
resolve_at: ${r.resolveAt}
requested_at: ${Math.floor(e.requestedAt)}
`}function d(e){let t=a(e.walletAddress),r=l(e.bootstrap);return`FunnyOption Operator Authorization

action: ISSUE_FIRST_LIQUIDITY
wallet: ${t}
market_id: ${r.marketId}
user_id: ${r.userId}
quantity: ${r.quantity}
outcome: ${r.outcome}
price: ${r.price}
requested_at: ${Math.floor(e.requestedAt)}
`}function c(e){let t=a(e.walletAddress),r=i(e.market);return`FunnyOption Operator Authorization

action: RESOLVE_MARKET
wallet: ${t}
market_id: ${r.marketId}
outcome: ${r.outcome}
requested_at: ${Math.floor(e.requestedAt)}
`}},7598:e=>{"use strict";e.exports=require("node:crypto")},8356:()=>{},9034:(e,t,r)=>{"use strict";r.d(t,{M0:()=>u,VR:()=>i,kR:()=>l,r2:()=>n});var o=r(4257),s=r(2554),a=r(6135);function n(){let e=Number(process.env.FUNNYOPTION_DEFAULT_OPERATOR_USER_ID??"1001"??"1001");return Number.isFinite(e)&&e>0?e:1001}async function u(e,t){let r=(0,a._X)(process.env.FUNNYOPTION_OPERATOR_WALLETS??""??"");if(0===r.length)return{ok:!1,response:o.NextResponse.json({error:"FUNNYOPTION_OPERATOR_WALLETS is not configured for the admin service"},{status:403})};let n=(0,a.Qh)(t.walletAddress);if(!n||!t.signature.trim()||t.requestedAt<=0)return{ok:!1,response:o.NextResponse.json({error:"operator wallet, signature, and requested_at are required"},{status:400})};if(Math.abs(Date.now()-Math.floor(t.requestedAt))>a.iH)return{ok:!1,response:o.NextResponse.json({error:"operator signature expired"},{status:401})};let u="";try{u=(0,a.Qh)(await (0,s.Q)({message:e,signature:t.signature}))}catch{return{ok:!1,response:o.NextResponse.json({error:"invalid operator signature"},{status:401})}}return u!==n?{ok:!1,response:o.NextResponse.json({error:"operator signature does not match wallet"},{status:401})}:r.includes(u)?{ok:!0,walletAddress:u}:{ok:!1,response:o.NextResponse.json({error:"wallet is not authorized for operator actions"},{status:403})}}function i(e){return{wallet_address:e.walletAddress,requested_at:e.requestedAt,signature:e.signature}}async function l(e,t){let r=await fetch(`http://127.0.0.1:8080${e}`,{method:t.method,headers:{"Content-Type":"application/json"},cache:"no-store",body:JSON.stringify(t.body)}),s=await r.json().catch(()=>null);return r.ok?{ok:!0,status:r.status,payload:s??{}}:{ok:!1,status:r.status,error:String(s?.error??`HTTP ${r.status}`),response:o.NextResponse.json({error:s?.error??`HTTP ${r.status}`},{status:r.status})}}},9294:e=>{"use strict";e.exports=require("next/dist/server/app-render/work-async-storage.external.js")}};var t=require("../../../../../../webpack-runtime.js");t.C(e);var r=e=>t(t.s=e),o=t.X(0,[343,448,554],()=>r(3457));module.exports=o})();