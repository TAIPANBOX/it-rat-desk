/* IT-RAT desk agent, LOCAL TEST BUILD (served by desk-test on localhost only).
   With the local /api/ask server present, the chat is a plain conversation
   with an assistant that answers from the site's own pages, through one of
   the models switchable in the header. Writing to the team lives on the email
   address beside the chat: pointing at it opens a short note form that hands
   the draft to the visitor's own mail client. Without the server it behaves
   exactly like the published concierge.js. */
(function(){
"use strict";
const MAIL="itratmail@gmail.com";
const reduce=matchMedia("(prefers-reduced-motion: reduce)").matches;
const esc=s=>String(s).replace(/[&<>"]/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c]));

const CSS=`.cg-mic{width:42px;flex:0 0 auto;border-radius:10px;border:1px solid var(--line-2);background:var(--panel-2);color:var(--dim);
  cursor:pointer;display:flex;align-items:center;justify-content:center}
.cg-mic svg{display:inline-block}
.cg-mic:hover{border-color:var(--accent);color:var(--fg)}
.cg-mic.on{color:#fff;border-color:var(--ember);background:color-mix(in srgb,var(--ember) 30%,var(--panel-2));animation:cgMic 1.2s ease-in-out infinite}
@keyframes cgMic{50%{box-shadow:0 0 0 6px color-mix(in srgb,var(--ember) 25%,transparent)}}
.cg-lang{flex:0 0 auto;align-self:center;font:600 10.5px var(--font-m);letter-spacing:.06em;color:var(--faint);background:none;border:0;cursor:pointer;padding:0 2px}
.cg-lang:hover{color:var(--fg)}
@media (prefers-reduced-motion: reduce){.cg-mic.on{animation:none}}
.cg-msg a{color:var(--fg);border-bottom:1px solid var(--line-2);text-decoration:none}
.cg-msg ul{margin:6px 0 2px;padding-left:18px}
.cg-msg li{margin:2px 0}
.cg-msg code{font:12px var(--font-m)}
.cg-meta{display:block;margin-top:8px;font:11px var(--font-m);color:var(--faint)}
.mp{position:absolute;z-index:60;width:min(360px,calc(100vw - 32px));padding:14px 14px 12px;border:1px solid var(--line-2);border-radius:var(--rad-s);
  background:var(--panel);box-shadow:0 18px 50px rgba(0,0,0,.45);font:13.5px/1.45 var(--font-t);color:var(--dim)}
.mp[hidden]{display:none}
.mp h3{margin:0 0 6px;font:600 14px var(--font-t);color:var(--fg);letter-spacing:0}
.mp label{display:block;margin:7px 0 3px;font:600 10.5px var(--font-m);text-transform:uppercase;letter-spacing:.06em;color:var(--faint)}
.mp input,.mp textarea,.mp select{width:100%;box-sizing:border-box;background:var(--panel-2);border:1px solid var(--line);border-radius:10px;
  padding:7px 10px;color:var(--fg);font:13.5px var(--font-t);outline:none}
.mp textarea{min-height:62px;resize:vertical}
.mp input:focus,.mp textarea:focus,.mp select:focus{border-color:color-mix(in srgb,var(--accent) 55%,var(--line-2))}
.mp-row{display:flex;gap:8px;margin-top:10px;flex-wrap:wrap}
.mp-go,.mp-copy{font:600 12.5px var(--font-t);padding:8px 13px;border-radius:999px;border:1px solid var(--line-2);cursor:pointer;text-decoration:none;color:var(--fg);background:var(--panel-2)}
.mp-go{border-color:var(--accent);background:color-mix(in srgb,var(--accent) 18%,var(--panel-2))}
.mp-go[aria-disabled="true"]{opacity:.45;pointer-events:none}
.mp-x{position:absolute;top:8px;right:10px;background:none;border:0;color:var(--faint);font-size:18px;cursor:pointer;line-height:1}
.mp-note{margin:6px 0 0;font-size:11.5px;color:var(--faint)}
.mp-say{margin:6px 0 0;font-size:12.5px;color:var(--fg)}
.mp-say:empty{display:none}
a.mp-anchor.mp-pulse{animation:mpPulse 1.6s ease-out 3}
@keyframes mpPulse{0%{box-shadow:0 0 0 0 color-mix(in srgb,var(--accent) 70%,transparent)}100%{box-shadow:0 0 0 10px transparent}}
@media (prefers-reduced-motion: reduce){a.mp-anchor.mp-pulse{animation:none}}`;

/* Escape first, then allow a small markdown subset. Links only to it-rat.com
   and to the TAIPANBOX repositories the pages themselves link to. */
function md(src){
  const inline=s=>esc(s)
    .replace(/\[([^\]]+)\]\((https:\/\/(?:it-rat\.com|github\.com\/TAIPANBOX)\/[^\s)]*)\)/g,'<a href="$2" target="_blank" rel="noopener">$1</a>')
    .replace(/\*\*([^*]+)\*\*/g,"<b>$1</b>")
    .replace(/`([^`]+)`/g,"<code>$1</code>");
  const out=[]; let list=null, code=null;
  for(const line of String(src).split("\n")){
    /* A fenced block is shown as typed: escaped, never formatted. */
    if(/^\s*```/.test(line)){
      if(code===null){ if(list){ out.push("<ul>"+list.join("")+"</ul>"); list=null; } code=[]; }
      else{ out.push("<pre class='cg-draft'><code>"+esc(code.join("\n"))+"</code></pre>"); code=null; }
      continue;
    }
    if(code!==null){ code.push(line); continue; }
    const m=line.match(/^\s*[-*]\s+(.*)$/);
    if(m){ if(!list){list=[];} list.push("<li>"+inline(m[1])+"</li>"); continue; }
    if(list){ out.push("<ul>"+list.join("")+"</ul>"); list=null; }
    out.push(line.trim()===""?"<br>":inline(line)+"<br>");
  }
  if(list) out.push("<ul>"+list.join("")+"</ul>");
  /* An unclosed fence is a stream still arriving: show what is there. */
  if(code!==null) out.push("<pre class='cg-draft'><code>"+esc(code.join("\n"))+"</code></pre>");
  return out.join("").replace(/(<br>)+$/,"").replace(/(<br>){3,}/g,"<br><br>");
}

/* ---- the note form on the email address ---- */
function mailPopover(link){
  const pop=document.createElement("div");
  pop.className="mp"; pop.hidden=true; pop.setAttribute("role","dialog"); pop.setAttribute("aria-label","Write to the team");
  pop.innerHTML=
    `<button type="button" class="mp-x" aria-label="Close">&times;</button>
     <h3>Write to the team</h3>
     <label for="mp-msg">Your message</label>
     <textarea id="mp-msg" placeholder="your setup, your question, what you want to try"></textarea>
     <label for="mp-who">Your name, and company if relevant</label>
     <input id="mp-who" type="text" autocomplete="name">
     <label for="mp-mail">Email for the reply</label>
     <input id="mp-mail" type="email" autocomplete="email" placeholder="you@company.com">
     <label for="mp-to">Who should read it first</label>
     <select id="mp-to"><option>Whoever fits</option><option value="Yurii">Yurii, cloud security and IAM</option><option value="Tania">Tania, FinOps and cost governance</option></select>
     <div class="mp-row"><a class="mp-go" aria-disabled="true">Open in my mail app</a><button type="button" class="mp-copy">Copy the note</button></div>
     <p class="mp-say" aria-live="polite"></p>
     <p class="mp-note">Nothing leaves this page: your own mail app opens with the note, addressed to ${esc(MAIL)}.</p>`;
  document.body.appendChild(pop);
  const $=s=>pop.querySelector(s);
  const msg=$("#mp-msg"), who=$("#mp-who"), mail=$("#mp-mail"), to=$("#mp-to"), go=$(".mp-go"), say=$(".mp-say");
  link.classList.add("mp-anchor");
  link.setAttribute("aria-haspopup","dialog"); link.setAttribute("aria-expanded","false");

  const okMail=v=>/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v.trim());
  function draft(){
    return "For: "+to.value+"\n"
      +"From: "+(who.value.trim()||"(no name given)")+"\n"
      +"Reply to: "+mail.value.trim()+"\n\n"
      +msg.value.trim()+"\n\n"
      +"(sent via the it-rat site desk agent)";
  }
  function refresh(){
    const ready=msg.value.trim()!==""&&okMail(mail.value);
    go.setAttribute("aria-disabled",ready?"false":"true");
    if(ready){
      const first=msg.value.trim().split("\n")[0].slice(0,60);
      go.href="mailto:"+MAIL+"?subject="+encodeURIComponent("IT-RAT inquiry for "+(to.value==="Whoever fits"?"the team":to.value)+": "+first)
        +"&body="+encodeURIComponent(draft());
    }else go.removeAttribute("href");
    say.textContent=ready?"":(msg.value.trim()===""?"":(mail.value.trim()===""?"Add an email for the reply.":(okMail(mail.value)?"":"That email does not look right yet.")));
  }
  [msg,who,mail,to].forEach(el=>el.addEventListener("input",refresh));

  function place(){
    /* Under the text column the address sits in, so the chat beside it stays clear. */
    const r=link.getBoundingClientRect(), w=pop.offsetWidth, vw=document.documentElement.clientWidth;
    const col=(link.closest("p")||link).getBoundingClientRect();
    let left=window.scrollX+col.left; left=Math.max(window.scrollX+16,Math.min(left,window.scrollX+vw-16-w));
    let top=window.scrollY+r.bottom+10;
    const below=window.innerHeight-r.bottom, h=pop.offsetHeight;
    if(below<h+20&&r.top>h+20) top=window.scrollY+r.top-h-10;
    pop.style.left=left+"px"; pop.style.top=top+"px";
  }
  let closeT=0;
  function open(focus){
    clearTimeout(closeT);
    if(pop.hidden){ pop.hidden=false; link.setAttribute("aria-expanded","true"); }
    place(); refresh();
    if(focus) setTimeout(()=>msg.focus({preventScroll:true}),30);
  }
  function close(){ pop.hidden=true; link.setAttribute("aria-expanded","false"); }
  /* Closing hands focus back to the address, and that focus must not reopen
     the form: Esc and the close button reopened it until 2026-11-01. */
  let returning=false;
  function closeAndReturn(){ close(); returning=true; link.focus({preventScroll:true}); returning=false; }
  /* A pointer leaving closes it, unless the visitor is typing or has typed. */
  function softClose(){
    clearTimeout(closeT);
    closeT=setTimeout(()=>{
      if(pop.contains(document.activeElement)) return;
      if(msg.value.trim()||who.value.trim()||mail.value.trim()) return;
      close();
    },350);
  }
  if(matchMedia("(hover: hover)").matches){
    link.addEventListener("mouseenter",()=>open(false));
    link.addEventListener("mouseleave",softClose);
    pop.addEventListener("mouseenter",()=>clearTimeout(closeT));
    pop.addEventListener("mouseleave",softClose);
    link.addEventListener("focus",()=>{ if(!returning) open(false); });
  }else{
    /* A touch screen has no hover, and a tap still fires mouseenter and focus
       before its click, so those must not be listened to here: they opened the
       form and the click then toggled it shut. A tap opens it; the close
       button or a tap outside closes it. */
    link.addEventListener("click",e=>{ e.preventDefault(); open(true); });
  }
  $(".mp-x").addEventListener("click",closeAndReturn);
  document.addEventListener("keydown",e=>{ if(e.key==="Escape"&&!pop.hidden) closeAndReturn(); });
  document.addEventListener("pointerdown",e=>{ if(!pop.hidden&&!pop.contains(e.target)&&e.target!==link) close(); });
  window.addEventListener("resize",()=>{ if(!pop.hidden) place(); });
  $(".mp-copy").addEventListener("click",()=>{
    (navigator.clipboard?navigator.clipboard.writeText(draft()):Promise.reject())
      .then(()=>{ say.textContent="Copied. Paste it to "+MAIL+" whenever suits."; })
      .catch(()=>{ say.textContent="Could not reach the clipboard. The address is "+MAIL+"."; });
  });
  go.addEventListener("click",()=>{ if(go.getAttribute("aria-disabled")!=="true") say.textContent="Handed to your mail app. A human reads these."; });

  return {
    prefill(text){
      if(text&&!msg.value.trim()){ msg.value=text; refresh(); }
      link.classList.remove("mp-pulse"); void link.offsetWidth; link.classList.add("mp-pulse");
    }
  };
}

function mount(root){
  root.classList.add("cg");
  root.innerHTML=
    `<div class="cg-head mono"><span class="cg-dot"></span><span class="cg-ht">desk-agent v2 · deterministic · no model in this path</span></div>
     <div class="cg-log" aria-live="polite"></div>
     <div class="cg-quick"></div>
     <form class="cg-form">
       <input type="text" autocomplete="off" spellcheck="false" placeholder="" aria-label="Your message">
       <button type="submit" class="cg-send" aria-label="Send">&#8594;</button>
     </form>`;
  const log=root.querySelector(".cg-log"), quick=root.querySelector(".cg-quick");
  const form=root.querySelector(".cg-form"), input=form.querySelector("input");
  const head=root.querySelector(".cg-ht");
  const note={intent:"",lines:[],to:"",who:"",email:""};
  let step="intent";
  let provider="", history=[], busy=false, mailForm=null;

  function add(cls,html){
    const d=document.createElement("div");
    d.className="cg-msg "+cls; d.innerHTML=html;
    log.appendChild(d); log.scrollTop=log.scrollHeight;
    return d;
  }
  function agent(html,then){
    const d=add("a","<i></i><i></i><i></i>");
    d.classList.add("typing");
    setTimeout(()=>{
      d.classList.remove("typing"); d.innerHTML=html;
      log.scrollTop=log.scrollHeight;
      if(then) then();
    },reduce?30:520);
  }
  function chips(list){
    quick.innerHTML=list.map(c=>`<button type="button" class="cg-chip" data-v="${esc(c)}">${esc(c)}</button>`).join("");
  }
  function hint(placeholder){ input.placeholder=placeholder; setTimeout(()=>input.focus({preventScroll:true}),80); }

  /* ---- assistant mode: a plain conversation on one model, no buttons ---- */
  function label(p){
    if(p.id==="haiku") return "Haiku 5.5";
    return p.model.split("/").pop().replace(":free","").split("-").slice(0,2).join("-")+" · free";
  }
  function trimmedHistory(){
    let h=history.slice(-19);
    if(h.length&&h[0].role!=="user") h=h.slice(1);
    return h;
  }
  /* Land on the first line of the answer, not its last: a long answer read
     from the bottom has to be scrolled back up to be read at all. */
  function toTop(d){ log.scrollTop+=d.getBoundingClientRect().top-log.getBoundingClientRect().top-10; }

  async function ask(q){
    busy=true; input.disabled=true;
    history.push({role:"user",text:q});
    const d=add("a typing","<i></i><i></i><i></i>");
    let text="", started=false, r=null, err="", raf=0;
    const paint=()=>{ raf=0; d.innerHTML=md(text); };
    /* The server sends one JSON event per line: status while a page is read,
       delta as the answer is written, reset when what streamed was narration,
       and done with the final answer, which is drawn over whatever streamed. */
    function on(ev){
      if(ev.t==="status"&&!started){
        d.classList.remove("typing");
        d.innerHTML=`<span class="cg-meta" style="margin-top:0">reading ${esc(String(ev.page||"").replace("https://it-rat.com/",""))}&hellip;</span>`;
      }else if(ev.t==="delta"||ev.t==="reset"){
        text=ev.t==="delta"?text+ev.text:(ev.text||"");
        if(!started&&text){ started=true; d.classList.remove("typing"); paint(); toTop(d); }
        else if(started&&!raf) raf=requestAnimationFrame(paint);
      }else if(ev.t==="done") r=ev.done;
    }
    try{
      const res=await fetch("/api/ask",{method:"POST",headers:{"Content-Type":"application/json"},
        body:JSON.stringify({provider,history:trimmedHistory()})});
      if(!res.ok){ const j=await res.json().catch(()=>({})); err=j.error||("http "+res.status); }
      else{
        const reader=res.body.getReader(), dec=new TextDecoder(); let buf="";
        for(;;){
          const {value,done}=await reader.read(); if(done) break;
          buf+=dec.decode(value,{stream:true});
          let nl; while((nl=buf.indexOf("\n"))>=0){ const line=buf.slice(0,nl).trim(); buf=buf.slice(nl+1); if(line) on(JSON.parse(line)); }
        }
      }
      if(!err&&(!r||r.error||!r.answer)) err=(r&&r.error)||"no answer";
    }catch(e){ err=String(e&&e.message||e); }
    if(raf) cancelAnimationFrame(raf);
    busy=false; input.disabled=false; d.classList.remove("typing");
    if(err){
      history.pop();
      d.innerHTML="The assistant did not answer this time ("+esc(err)+"). Ask again in a moment.";
      log.scrollTop=log.scrollHeight; hint("ask again"); return;
    }
    history.push({role:"assistant",text:r.answer});
    if(r.offer_contact&&mailForm) mailForm.prefill(r.contact_summary||q);
    const pages=(r.pages_read||[]).length;
    const cost=r.provider==="haiku"?("$"+r.cost_usd.toFixed(4)):"$0";
    const first=r.first_ms?` · first words ${(r.first_ms/1000).toFixed(1)}s`:"";
    d.innerHTML=md(r.answer)+`<span class="cg-meta">${esc(label({id:r.provider,model:r.model}))} · ${pages} page${pages===1?"":"s"} read${first} · ${(r.ms/1000).toFixed(1)}s · ${cost}</span>`;
    if(!started) toTop(d);
    hint("ask another question");
  }

  /* ---- voice input through the browser's own speech recognition ---- */
  function mountMic(){
    const SR=window.SpeechRecognition||window.webkitSpeechRecognition;
    if(!SR) return;
    let lang=(navigator.languages||[navigator.language||""]).some(l=>/^uk/i.test(l))?"uk-UA":"en-US";
    const tag=document.createElement("button"); tag.type="button"; tag.className="cg-lang";
    const mic=document.createElement("button"); mic.type="button"; mic.className="cg-mic";
    mic.innerHTML='<svg viewBox="0 0 24 24" width="17" height="17" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/></svg>';
    const label=()=>{ tag.textContent=lang==="uk-UA"?"UA":"EN"; tag.title="Speech language: "+lang+" (click to switch)"; mic.setAttribute("aria-label","Speak ("+lang+")"); mic.title="Speak ("+lang+")"; };
    label();
    form.insertBefore(tag,form.querySelector(".cg-send"));
    form.insertBefore(mic,form.querySelector(".cg-send"));
    tag.addEventListener("click",()=>{ if(rec) return; lang=lang==="uk-UA"?"en-US":"uk-UA"; label(); });
    let rec=null;
    mic.addEventListener("click",()=>{
      if(rec){ rec.stop(); return; }
      if(busy) return;
      rec=new SR(); rec.lang=lang; rec.interimResults=true; rec.continuous=false; rec.maxAlternatives=1;
      const base=input.value.trim()?input.value.trimEnd()+" ":"";
      rec.onresult=e=>{ let t=""; for(let i=0;i<e.results.length;i++) t+=e.results[i][0].transcript; input.value=base+t; };
      rec.onerror=e=>{
        input.placeholder=e.error==="not-allowed"||e.error==="service-not-allowed"?"microphone blocked: allow it for this page"
          :e.error==="network"?"this browser cannot recognise speech here: try Chrome or Safari"
          :e.error==="no-speech"?"heard nothing, try again":"did not catch that, try again";
      };
      rec.onend=()=>{ rec=null; mic.classList.remove("on"); tag.disabled=false; input.focus({preventScroll:true}); };
      mic.classList.add("on"); tag.disabled=true;
      try{ rec.start(); }catch(e){ rec=null; mic.classList.remove("on"); tag.disabled=false; }
    });
  }

  /* ---- the published deterministic flow, used when no local server answers ---- */
  function draft(){
    return "For: "+(note.to||"whoever fits")+"\n"
      +"Intent: "+note.intent+"\n"
      +"From: "+note.who+"\n"
      +"Reply to: "+note.email+"\n\n"
      +note.lines.join("\n")+"\n\n"
      +"(sent via the it-rat site desk agent)";
  }
  function mailtoHref(){
    return "mailto:"+MAIL
      +"?subject="+encodeURIComponent("IT-RAT inquiry for "+(note.to||"the team")+": "+note.intent)
      +"&body="+encodeURIComponent(draft());
  }
  function askRecipient(){
    step="to";
    agent("Who should read it first? Yurii lives in cloud security and IAM, Tania in FinOps and cost governance. You can also just name a topic.",
      ()=>{chips(["Yurii","Tania","Whoever fits"]);hint("or type a name / topic");});
  }
  function askWho(){
    step="who";
    agent("Who should the reply go to? Name, and company if relevant.",()=>{quick.innerHTML="";hint("name, company");});
  }
  const onText={
    ask(v){ ask(v); },
    intent(v){
      note.intent="Direct note"; note.lines.push(v); step="details";
      agent("Noted, word for word. Add as much as you like, every message becomes a line of the note. Done? Say so or press the button.",
        ()=>{chips(["That covers it"]);hint("add another sentence, or send the chip");});
    },
    details(v){ note.lines.push(v); chips(["That covers it"]);hint("anything else?"); },
    to(v){ note.to=v; askWho(); },
    who(v){
      note.who=v; step="email";
      agent("And an email for the answer. It is only used to write back, nothing else.",()=>hint("you@company.com"));
    },
    email(v){
      if(!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v.trim())){
        agent("That does not parse as an email. One more try?",()=>hint("you@company.com"));
        return;
      }
      note.email=v.trim(); step="done"; input.placeholder="";
      agent("Here is the note, exactly as it will leave this page:"
        +"<pre class='cg-draft'>"+esc(draft())+"</pre>"
        +"Press send and your own mail client opens with this text. This page keeps no copy.",
        ()=>chips(["Send from my mail app","Copy the note","Start over"]));
    }
  };
  quick.addEventListener("click",e=>{
    const b=e.target.closest(".cg-chip"); if(!b||busy) return;
    const v=b.dataset.v;
    if(step==="intent"){
      add("u",esc(v)); note.intent=v; step="details";
      agent("Good. Tell me about it in your own words: your setup, your question, what you want to try. Several messages are fine.",
        ()=>{chips(["That covers it"]);hint("a sentence or two, or a few");});
      return;
    }
    if(step==="details"&&v==="That covers it"){
      if(!note.lines.length){ agent("Give me at least one sentence to carry, otherwise the note travels empty.",()=>hint("a sentence or two")); return; }
      add("u","that covers it"); askRecipient(); return;
    }
    if(step==="to"){ add("u",esc(v)); note.to=v; askWho(); return; }
    if(step==="done"){
      if(v==="Send from my mail app"){ add("u","send"); location.href=mailtoHref(); agent("Handed to your mail client. Thank you, a human reads these: "+(note.to&&note.to!=="Whoever fits"?note.to:"one of the two of us")+"."); }
      else if(v==="Copy the note"){
        add("u","copy");
        (navigator.clipboard?navigator.clipboard.writeText(draft()):Promise.reject())
          .then(()=>agent("Copied. Paste it to "+MAIL+" whenever suits."))
          .catch(()=>agent("Could not reach the clipboard. Select the draft above and copy it, the address is "+MAIL+"."));
      }
      else if(v==="Start over"){
        log.innerHTML=""; quick.innerHTML="";
        note.intent="";note.lines=[];note.to="";note.who="";note.email="";
        bootDeterministic();
      }
    }
  });
  form.addEventListener("submit",e=>{
    e.preventDefault();
    const v=input.value.trim(); if(!v||step==="done"||busy) return;
    add("u",esc(v)); input.value="";
    onText[step](v);
  });

  function bootDeterministic(){
    step="intent";
    agent("Desk agent online. Deterministic on purpose, like the rest of the decision path: no model here, no tracking, and nothing leaves this page until you press send. Pick a shortcut, or just start typing.",
      ()=>{chips(["Deploy the stack","Consulting","Something else"]);hint("write anything, I listen");});
  }
  function bootAssistant(){
    step="ask"; quick.innerHTML="";
    agent("Desk assistant, test build. Ask anything about this site: I read its pages and answer with a link to where I read it.",
      ()=>hint("ask about the stack, the guides, the team"));
  }

  const ctl=new AbortController(); const tm=setTimeout(()=>ctl.abort(),1500);
  fetch("/api/health",{signal:ctl.signal}).then(r=>r.ok?r.json():null).catch(()=>null).then(h=>{
    clearTimeout(tm);
    if(h&&h.ok&&h.providers&&h.providers.length){
      provider=(h.providers.find(p=>p.id==="haiku")||h.providers[0]).id;
      const st=document.createElement("style"); st.textContent=CSS; document.head.appendChild(st);
      head.textContent="desk-agent v3 · test build · answers from this site";
      mountMic();
      /* The note to the team moves onto the address beside the chat. */
      const link=root.closest(".foot-cta")?.querySelector('a[href^="mailto:"]');
      if(link){
        const lede=link.closest("p");
        if(lede){
          lede.textContent="";
          lede.append("A question about the stack, an idea for a pilot, or just a hello. The assistant answers from this site's own pages. To write to the team, point at the address: ",link);
        }
        mailForm=mailPopover(link);
      }
      bootAssistant();
    }else bootDeterministic();
  });
}

const el=document.querySelector("[data-concierge]");
if(el) mount(el);
})();
