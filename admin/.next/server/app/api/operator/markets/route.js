(()=>{var e={};e.id=316,e.ids=[316],e.modules={846:e=>{"use strict";e.exports=require("next/dist/compiled/next-server/app-page.runtime.prod.js")},3033:e=>{"use strict";e.exports=require("next/dist/server/app-render/work-unit-async-storage.external.js")},3295:e=>{"use strict";e.exports=require("next/dist/server/app-render/after-task-async-storage.external.js")},4870:e=>{"use strict";e.exports=require("next/dist/compiled/next-server/app-route.runtime.prod.js")},5620:()=>{},6135:(e,t,r)=>{"use strict";r.d(t,{P4:()=>i,Qh:()=>a,WZ:()=>c,Xt:()=>l,_X:()=>n,cD:()=>d,iH:()=>o,ph:()=>u,wy:()=>p});let o=3e5;function s(e){return e.trim().replace(/\s+/g," ")}function a(e){return e.trim().toLowerCase()}function n(e){return e.split(",").map(e=>a(e)).filter(Boolean)}function u(e){return{title:s(e.title),description:s(e.description),category:s(e.category)||"Polymarket",coverImage:e.coverImage.trim(),sourceUrl:e.sourceUrl.trim(),sourceSlug:s(e.sourceSlug),sourceName:s(e.sourceName)||"Polymarket",sourceKind:s(e.sourceKind).toLowerCase()||"manual",status:s(e.status).toUpperCase()||"OPEN",collateralAsset:s(e.collateralAsset).toUpperCase()||"USDT",openAt:Math.max(0,Math.floor(e.openAt||0)),closeAt:Math.max(0,Math.floor(e.closeAt||0)),resolveAt:Math.max(0,Math.floor(e.resolveAt||0))}}function i(e){return{marketId:Math.max(0,Math.floor(e.marketId||0)),outcome:"NO"===s(e.outcome).toUpperCase()?"NO":"YES"}}function l(e){return{marketId:Math.max(0,Math.floor(e.marketId||0)),userId:Math.max(0,Math.floor(e.userId||0)),quantity:Math.max(0,Math.floor(e.quantity||0)),outcome:"NO"===s(e.outcome).toUpperCase()?"NO":"YES",price:Math.max(0,Math.floor(e.price||0))}}function c(e){let t=a(e.walletAddress),r=u(e.market);return`FunnyOption Operator Authorization

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
`}function p(e){let t=a(e.walletAddress),r=l(e.bootstrap);return`FunnyOption Operator Authorization

action: ISSUE_FIRST_LIQUIDITY
wallet: ${t}
market_id: ${r.marketId}
user_id: ${r.userId}
quantity: ${r.quantity}
outcome: ${r.outcome}
price: ${r.price}
requested_at: ${Math.floor(e.requestedAt)}
`}function d(e){let t=a(e.walletAddress),r=i(e.market);return`FunnyOption Operator Authorization

action: RESOLVE_MARKET
wallet: ${t}
market_id: ${r.marketId}
outcome: ${r.outcome}
requested_at: ${Math.floor(e.requestedAt)}
`}},6588:(e,t,r)=>{"use strict";r.r(t),r.d(t,{patchFetch:()=>A,routeModule:()=>p,serverHooks:()=>h,workAsyncStorage:()=>d,workUnitAsyncStorage:()=>m});var o={};r.r(o),r.d(o,{POST:()=>c});var s=r(4296),a=r(8093),n=r(8156),u=r(4257),i=r(6135),l=r(9034);async function c(e){let t=await e.json().catch(()=>null);if(!t?.market||!t?.operator)return u.NextResponse.json({error:"market and operator payloads are required"},{status:400});let r=(0,i.ph)(t.market);if(!r.title)return u.NextResponse.json({error:"title is required"},{status:400});let o=await (0,l.M0)((0,i.WZ)({walletAddress:t.operator.walletAddress,market:r,requestedAt:t.operator.requestedAt}),t.operator);if(!o.ok)return o.response;let s=await (0,l.kR)("/api/v1/markets",{method:"POST",body:{title:r.title,description:r.description,collateral_asset:r.collateralAsset,status:r.status,open_at:r.openAt,close_at:r.closeAt,resolve_at:r.resolveAt,created_by:(0,l.r2)(),cover_image_url:r.coverImage,cover_source_url:r.sourceUrl,cover_source_name:r.sourceName,metadata:{category:r.category,coverImage:r.coverImage,sourceUrl:r.sourceUrl,sourceSlug:r.sourceSlug,sourceName:r.sourceName,sourceKind:r.sourceKind,yesOdds:.5,noOdds:.5},operator:(0,l.VR)(t.operator)}});return s.ok?u.NextResponse.json({...s.payload,operator_wallet_address:o.walletAddress},{status:s.status}):s.response}let p=new s.AppRouteRouteModule({definition:{kind:a.RouteKind.APP_ROUTE,page:"/api/operator/markets/route",pathname:"/api/operator/markets",filename:"route",bundlePath:"app/api/operator/markets/route"},resolvedPagePath:"/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/route.ts",nextConfigOutput:"",userland:o}),{workAsyncStorage:d,workUnitAsyncStorage:m,serverHooks:h}=p;function A(){return(0,n.patchFetch)({workAsyncStorage:d,workUnitAsyncStorage:m})}},7598:e=>{"use strict";e.exports=require("node:crypto")},8356:()=>{},9034:(e,t,r)=>{"use strict";r.d(t,{M0:()=>u,VR:()=>i,kR:()=>l,r2:()=>n});var o=r(4257),s=r(2554),a=r(6135);function n(){let e=Number(process.env.FUNNYOPTION_DEFAULT_OPERATOR_USER_ID??"1001"??"1001");return Number.isFinite(e)&&e>0?e:1001}async function u(e,t){let r=(0,a._X)(process.env.FUNNYOPTION_OPERATOR_WALLETS??""??"");if(0===r.length)return{ok:!1,response:o.NextResponse.json({error:"FUNNYOPTION_OPERATOR_WALLETS is not configured for the admin service"},{status:403})};let n=(0,a.Qh)(t.walletAddress);if(!n||!t.signature.trim()||t.requestedAt<=0)return{ok:!1,response:o.NextResponse.json({error:"operator wallet, signature, and requested_at are required"},{status:400})};if(Math.abs(Date.now()-Math.floor(t.requestedAt))>a.iH)return{ok:!1,response:o.NextResponse.json({error:"operator signature expired"},{status:401})};let u="";try{u=(0,a.Qh)(await (0,s.Q)({message:e,signature:t.signature}))}catch{return{ok:!1,response:o.NextResponse.json({error:"invalid operator signature"},{status:401})}}return u!==n?{ok:!1,response:o.NextResponse.json({error:"operator signature does not match wallet"},{status:401})}:r.includes(u)?{ok:!0,walletAddress:u}:{ok:!1,response:o.NextResponse.json({error:"wallet is not authorized for operator actions"},{status:403})}}function i(e){return{wallet_address:e.walletAddress,requested_at:e.requestedAt,signature:e.signature}}async function l(e,t){let r=await fetch(`http://127.0.0.1:8080${e}`,{method:t.method,headers:{"Content-Type":"application/json"},cache:"no-store",body:JSON.stringify(t.body)}),s=await r.json().catch(()=>null);return r.ok?{ok:!0,status:r.status,payload:s??{}}:{ok:!1,status:r.status,error:String(s?.error??`HTTP ${r.status}`),response:o.NextResponse.json({error:s?.error??`HTTP ${r.status}`},{status:r.status})}}},9294:e=>{"use strict";e.exports=require("next/dist/server/app-render/work-async-storage.external.js")}};var t=require("../../../../webpack-runtime.js");t.C(e);var r=e=>t(t.s=e),o=t.X(0,[343,448,554],()=>r(6588));module.exports=o})();