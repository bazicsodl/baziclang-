; Bazic LLVM IR (early backend)
source_filename = "bazic_module"

declare i32 @printf(ptr, ...)
declare i64 @strlen(ptr)
declare i32 @strcmp(ptr, ptr)
declare ptr @strstr(ptr, ptr)
declare i32 @strncmp(ptr, ptr, i64)
declare i32 @toupper(i32)
declare i32 @tolower(i32)
declare i32 @isspace(i32)
declare i64 @strtol(ptr, ptr, i32)
declare double @strtod(ptr, ptr)
declare i32 @snprintf(ptr, i64, ptr, ...)
declare ptr @malloc(i64)
declare ptr @memcpy(ptr, ptr, i64)

; struct Error
; struct User
; interface Named
; struct Box__int

%Error = type { ptr }
%User = type { ptr, i64 }
%Box__int = type { i64 }

%Named = type { ptr, ptr }

@.str0 = private unnamed_addr constant [4 x i8] c"%ld\00"
@.str1 = private unnamed_addr constant [5 x i8] c"%ld\0A\00"
@.str2 = private unnamed_addr constant [3 x i8] c"%g\00"
@.str3 = private unnamed_addr constant [4 x i8] c"%g\0A\00"
@.str4 = private unnamed_addr constant [3 x i8] c"%s\00"
@.str5 = private unnamed_addr constant [4 x i8] c"%s\0A\00"
@.str6 = private unnamed_addr constant [5 x i8] c"true\00"
@.str7 = private unnamed_addr constant [6 x i8] c"false\00"
@.str8 = private unnamed_addr constant [1 x i8] c"\00"
@.str9 = private unnamed_addr constant [12 x i8] c"invalid int\00"
@.str10 = private unnamed_addr constant [14 x i8] c"invalid float\00"
@.str11 = private unnamed_addr constant [16 x i8] c"std unavailable\00"
@.str12 = private unnamed_addr constant [17 x i8] c"assertion failed\00"
@.str13 = private unnamed_addr constant [5 x i8] c"Ipeh\00"

define ptr @bazic_str_concat(ptr %a, ptr %b) {
entry:
  %lenA = call i64 @strlen(ptr %a)
  %lenB = call i64 @strlen(ptr %b)
  %sum = add i64 %lenA, %lenB
  %total = add i64 %sum, 1
  %buf = call ptr @malloc(i64 %total)
  call ptr @memcpy(ptr %buf, ptr %a, i64 %lenA)
  %dstB = getelementptr i8, ptr %buf, i64 %lenA
  call ptr @memcpy(ptr %dstB, ptr %b, i64 %lenB)
  %end = getelementptr i8, ptr %buf, i64 %sum
  store i8 0, ptr %end
  ret ptr %buf
}

define i32 @bazic_str_cmp(ptr %a, ptr %b) {
entry:
  %c = call i32 @strcmp(ptr %a, ptr %b)
  ret i32 %c
}

define i1 @bazic_contains(ptr %s, ptr %sub) {
entry:
  %found = call ptr @strstr(ptr %s, ptr %sub)
  %ok = icmp ne ptr %found, null
  ret i1 %ok
}

define i1 @bazic_starts_with(ptr %s, ptr %prefix) {
entry:
  %len = call i64 @strlen(ptr %prefix)
  %cmp = call i32 @strncmp(ptr %s, ptr %prefix, i64 %len)
  %ok = icmp eq i32 %cmp, 0
  ret i1 %ok
}

define i1 @bazic_ends_with(ptr %s, ptr %suffix) {
entry:
  %lenS = call i64 @strlen(ptr %s)
  %lenT = call i64 @strlen(ptr %suffix)
  %short = icmp ult i64 %lenS, %lenT
  br i1 %short, label %retfalse, label %cont
retfalse:
  ret i1 0
cont:
  %start = sub i64 %lenS, %lenT
  %ptr = getelementptr i8, ptr %s, i64 %start
  %cmp = call i32 @strncmp(ptr %ptr, ptr %suffix, i64 %lenT)
  %ok = icmp eq i32 %cmp, 0
  ret i1 %ok
}

define ptr @bazic_to_upper(ptr s) {
entry:
  %len = call i64 @strlen(ptr %s)
  %total = add i64 %len, 1
  %buf = call ptr @malloc(i64 %total)
  br label %loop
loop:
  %i = phi i64 [ 0, %entry ], [ %next, %loop ]
  %done = icmp eq i64 %i, %len
  br i1 %done, label %end, label %body
body:
  %srcPtr = getelementptr i8, ptr %s, i64 %i
  %ch = load i8, ptr %srcPtr
  %ch32 = zext i8 %ch to i32
  %conv = call i32 @toupper(i32 %ch32)
  %out = trunc i32 %conv to i8
  %dstPtr = getelementptr i8, ptr %buf, i64 %i
  store i8 %out, ptr %dstPtr
  %next = add i64 %i, 1
  br label %loop
end:
  %endPtr = getelementptr i8, ptr %buf, i64 %len
  store i8 0, ptr %endPtr
  ret ptr %buf
}

define ptr @bazic_to_lower(ptr s) {
entry:
  %len = call i64 @strlen(ptr %s)
  %total = add i64 %len, 1
  %buf = call ptr @malloc(i64 %total)
  br label %loop
loop:
  %i = phi i64 [ 0, %entry ], [ %next, %loop ]
  %done = icmp eq i64 %i, %len
  br i1 %done, label %end, label %body
body:
  %srcPtr = getelementptr i8, ptr %s, i64 %i
  %ch = load i8, ptr %srcPtr
  %ch32 = zext i8 %ch to i32
  %conv = call i32 @tolower(i32 %ch32)
  %out = trunc i32 %conv to i8
  %dstPtr = getelementptr i8, ptr %buf, i64 %i
  store i8 %out, ptr %dstPtr
  %next = add i64 %i, 1
  br label %loop
end:
  %endPtr = getelementptr i8, ptr %buf, i64 %len
  store i8 0, ptr %endPtr
  ret ptr %buf
}

define ptr @bazic_trim_space(ptr %s) {
entry:
  %len = call i64 @strlen(ptr %s)
  %start = alloca i64
  store i64 0, ptr %start
  br label %loop_start
loop_start:
  %i = load i64, ptr %start
  %done = icmp uge i64 %i, %len
  br i1 %done, label %allspace, label %check_start
check_start:
  %ptr = getelementptr i8, ptr %s, i64 %i
  %ch = load i8, ptr %ptr
  %ch32 = zext i8 %ch to i32
  %is = call i32 @isspace(i32 %ch32)
  %iss = icmp ne i32 %is, 0
  br i1 %iss, label %inc_start, label %start_done
inc_start:
  %ni = add i64 %i, 1
  store i64 %ni, ptr %start
  br label %loop_start
start_done:
  %startVal = load i64, ptr %start
  %end = alloca i64
  %last = sub i64 %len, 1
  store i64 %last, ptr %end
  br label %loop_end
loop_end:
  %j = load i64, ptr %end
  %lt = icmp ult i64 %j, %startVal
  br i1 %lt, label %allspace, label %check_end
check_end:
  %ptr2 = getelementptr i8, ptr %s, i64 %j
  %ch2 = load i8, ptr %ptr2
  %ch32b = zext i8 %ch2 to i32
  %is2 = call i32 @isspace(i32 %ch32b)
  %iss2 = icmp ne i32 %is2, 0
  br i1 %iss2, label %dec_end, label %end_done
dec_end:
  %nj = sub i64 %j, 1
  store i64 %nj, ptr %end
  br label %loop_end
end_done:
  %endVal = load i64, ptr %end
  %newLen = sub i64 %endVal, %startVal
  %newLen2 = add i64 %newLen, 1
  %total = add i64 %newLen2, 1
  %buf = call ptr @malloc(i64 %total)
  %src = getelementptr i8, ptr %s, i64 %startVal
  call ptr @memcpy(ptr %buf, ptr %src, i64 %newLen2)
  %endPtr = getelementptr i8, ptr %buf, i64 %newLen2
  store i8 0, ptr %endPtr
  ret ptr %buf
allspace:
  %empty = getelementptr inbounds ([1 x i8], ptr @.str8, i64 0, i64 0)
  ret ptr %empty
}

define ptr @bazic_repeat(ptr %s, i64 %count) {
entry:
  %nonpos = icmp sle i64 %count, 0
  br i1 %nonpos, label %empty, label %cont
empty:
  %empty = getelementptr inbounds ([1 x i8], ptr @.str8, i64 0, i64 0)
  ret ptr %empty
cont:
  %len = call i64 @strlen(ptr %s)
  %total = mul i64 %len, %count
  %alloc = add i64 %total, 1
  %buf = call ptr @malloc(i64 %alloc)
  br label %loop
loop:
  %i = phi i64 [ 0, %cont ], [ %next, %loop ]
  %done = icmp eq i64 %i, %count
  br i1 %done, label %end, label %body
body:
  %offset = mul i64 %i, %len
  %dst = getelementptr i8, ptr %buf, i64 %offset
  call ptr @memcpy(ptr %dst, ptr %s, i64 %len)
  %next = add i64 %i, 1
  br label %loop
end:
  %endPtr = getelementptr i8, ptr %buf, i64 %total
  store i8 0, ptr %endPtr
  ret ptr %buf
}

define ptr @bazic_replace(ptr %s, ptr %old, ptr %new) {
entry:
  %oldLen = call i64 @strlen(ptr %old)
  %zero = icmp eq i64 %oldLen, 0
  br i1 %zero, label %retorig, label %count
retorig:
  ret ptr %s
count:
  %count = alloca i64
  store i64 0, ptr %count
  %cursor = alloca ptr
  store ptr %s, ptr %cursor
  br label %count_loop
count_loop:
  %cur = load ptr, ptr %cursor
  %found = call ptr @strstr(ptr %cur, ptr %old)
  %isnull = icmp eq ptr %found, null
  br i1 %isnull, label %count_done, label %count_hit
count_hit:
  %c = load i64, ptr %count
  %c1 = add i64 %c, 1
  store i64 %c1, ptr %count
  %next = getelementptr i8, ptr %found, i64 %oldLen
  store ptr %next, ptr %cursor
  br label %count_loop
count_done:
  %cfinal = load i64, ptr %count
  %noccur = icmp eq i64 %cfinal, 0
  br i1 %noccur, label %retorig, label %alloc
alloc:
  %lenS = call i64 @strlen(ptr %s)
  %lenN = call i64 @strlen(ptr %new)
  %diff = sub i64 %lenN, %oldLen
  %extra = mul i64 %diff, %cfinal
  %newLen = add i64 %lenS, %extra
  %total = add i64 %newLen, 1
  %buf = call ptr @malloc(i64 %total)
  %src = alloca ptr
  %dst = alloca ptr
  store ptr %s, ptr %src
  store ptr %buf, ptr %dst
  br label %loop
loop:
  %srcv = load ptr, ptr %src
  %found2 = call ptr @strstr(ptr %srcv, ptr %old)
  %isnull2 = icmp eq ptr %found2, null
  br i1 %isnull2, label %copy_tail, label %copy_seg
copy_seg:
  %srcInt = ptrtoint ptr %srcv to i64
  %foundInt = ptrtoint ptr %found2 to i64
  %segLen = sub i64 %foundInt, %srcInt
  %dstv = load ptr, ptr %dst
  call ptr @memcpy(ptr %dstv, ptr %srcv, i64 %segLen)
  %dst2 = getelementptr i8, ptr %dstv, i64 %segLen
  call ptr @memcpy(ptr %dst2, ptr %new, i64 %lenN)
  %dst3 = getelementptr i8, ptr %dst2, i64 %lenN
  store ptr %dst3, ptr %dst
  %nextSrc = getelementptr i8, ptr %found2, i64 %oldLen
  store ptr %nextSrc, ptr %src
  br label %loop
copy_tail:
  %srcv2 = load ptr, ptr %src
  %tailLen = call i64 @strlen(ptr %srcv2)
  %dstv2 = load ptr, ptr %dst
  call ptr @memcpy(ptr %dstv2, ptr %srcv2, i64 %tailLen)
  %end = getelementptr i8, ptr %dstv2, i64 %tailLen
  store i8 0, ptr %end
  ret ptr %buf
}

define ptr @bazic_int_to_str(i64 %v) {
entry:
  %fmt = getelementptr inbounds ([4 x i8], ptr @.str0, i64 0, i64 0)
  %len32 = call i32 (ptr, i64, ptr, ...) @snprintf(ptr null, i64 0, ptr %fmt, i64 %v)
  %len = sext i32 %len32 to i64
  %total = add i64 %len, 1
  %buf = call ptr @malloc(i64 %total)
  call i32 (ptr, i64, ptr, ...) @snprintf(ptr %buf, i64 %total, ptr %fmt, i64 %v)
  ret ptr %buf
}

define ptr @bazic_float_to_str(double %v) {
entry:
  %fmt = getelementptr inbounds ([3 x i8], ptr @.str2, i64 0, i64 0)
  %len32 = call i32 (ptr, i64, ptr, ...) @snprintf(ptr null, i64 0, ptr %fmt, double %v)
  %len = sext i32 %len32 to i64
  %total = add i64 %len, 1
  %buf = call ptr @malloc(i64 %total)
  call i32 (ptr, i64, ptr, ...) @snprintf(ptr %buf, i64 %total, ptr %fmt, double %v)
  ret ptr %buf
}


define i1 @__std_exists(ptr %path) { ret i1 0 }

define i64 @__std_unix_millis() { ret i64 0 }

define void @__std_sleep_ms(i64 %ms) { ret void }

define ptr @__std_now_rfc3339() {
entry:
  %empty = getelementptr inbounds ([1 x i8], ptr @.str8, i64 0, i64 0)
  ret ptr %empty
}

define ptr @__std_json_escape(ptr %s) { ret ptr %s }

define ptr @__std_sha256_hex(ptr %s) {
entry:
  %empty = getelementptr inbounds ([1 x i8], ptr @.str8, i64 0, i64 0)
  ret ptr %empty
}


define void @assert(i1 %cond) {
entry:
  %t1 = alloca i1
  store i1 %cond, ptr %t1
  %t2 = load i1, ptr %t1
  %t3 = xor i1 %t2, true
  br i1 %t3, label %then1, label %ifend3
then1:
ifend3:
  ret void
}

define void @assert_msg(i1 %cond, ptr %msg) {
entry:
  %t1 = alloca i1
  store i1 %cond, ptr %t1
  %t2 = alloca ptr
  store ptr %msg, ptr %t2
  %t3 = load i1, ptr %t1
  %t4 = xor i1 %t3, true
  br i1 %t4, label %then1, label %ifend3
then1:
ifend3:
  ret void
}

define ptr @User_label(%User %self) {
entry:
  %t1 = alloca %User
  store %User %self, ptr %t1
  %t2 = load %User, ptr %t1
  %t3 = extractvalue %User %t2, 0
  ret ptr %t3
}

define i32 @main() {
entry:
  %t1 = alloca %User
  %t2 = undef %User
  %t3 = insertvalue %User %t2, i64 27, 1
  %t4 = getelementptr inbounds ([5 x i8], ptr @.str13, i64 0, i64 0)
  %t5 = insertvalue %User %t3, ptr %t4, 0
  store %User %t5, ptr %t1
  %t6 = alloca %Box__int
  %t7 = undef %Box__int
  %t8 = insertvalue %Box__int %t7, i64 99, 0
  store %Box__int %t8, ptr %t6
  %t9 = load %User, ptr %t1
  %t10 = call ptr @User_label(%User %t9)
  %t11 = getelementptr inbounds ([4 x i8], ptr @.str5, i64 0, i64 0)
  call i32 @printf(ptr %t11, ptr %t10)
  %t12 = load %Box__int, ptr %t6
  %t13 = extractvalue %Box__int %t12, 0
  %t14 = getelementptr inbounds ([5 x i8], ptr @.str1, i64 0, i64 0)
  call i32 @printf(ptr %t14, i64 %t13)
  ret i32 0
}

