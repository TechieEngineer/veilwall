import test from 'node:test';import assert from 'node:assert/strict';import {evaluate,validateRules} from '../src/preview.js';
const r={id:'rule',name:'Rule',direction:'input',protocol:'tcp',source:'192.0.2.0/24',destination:'',port:443,action:'drop',mode:'shadow',priority:100};const p={direction:'input',protocol:'tcp',source:'192.0.2.10',destination:'198.51.100.1',port:443};
test('shadow is non-terminal',()=>{const out=evaluate([r],p);assert.equal(out.verdict,'accept');assert.deepEqual(out.shadow_matches,['rule'])});
test('active and priority',()=>{assert.equal(evaluate([{...r,mode:'active'}],p).verdict,'drop');assert.equal(evaluate([{...r,mode:'active',priority:2},{...r,id:'allow',mode:'active',action:'accept',priority:1}],p).verdict,'accept')});
test('IPv6 CIDR',()=>{assert.equal(evaluate([{...r,mode:'active',source:'2001:db8::/32'}],{...p,source:'2001:db8::1',destination:'2001:db8:1::2'}).verdict,'drop')});
test('validation rejects duplicate IDs and invalid addresses',()=>{assert.throws(()=>validateRules([r,r]));assert.throws(()=>validateRules([{...r,source:'300.1.2.3'}]));assert.throws(()=>validateRules([{...r,source:'2001:db8::/129'}]))});
