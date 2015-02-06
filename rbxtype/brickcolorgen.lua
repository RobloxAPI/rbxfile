-- This file generates the content of brickcolor.go

-- 1. Open Roblox Studio
-- 2. Create a new Place
-- 3. Run this file in the command bar or a Script
-- 4. Copy the generated data from the output

local MAX  = 1032 -- Maximum BrickColor number to check
local PREC = 18   -- Color3 float precision

local maxlen = #tostring(MAX)
local function align(bc)
	return string.rep(' ', maxlen - #tostring(bc.Number)+1)
end

local function msg(s)
	local t = string.rep('-', 16)
	print(string.format('%s %s %s', t, s:upper(), t))
end

local data = ''
local function write(f, ...)
	data = data .. string.format(f, ...)
end

write('package rbxtype\n')
write('\n')
write('// This file is automatically generated by brickcolorgen.lua\n')
write('\n')

write('var brickColorDefault = BrickColor(%d)\n', BrickColor.new(-1).Number)

write('\n')

local colors = {}
for i = 0, MAX do
	local bc = BrickColor.new(i)
	if bc.Number == i then
		colors[#colors+1] = bc
	end
end

write('var brickColorNames = map[BrickColor]string{\n')
for i = 1, #colors do
	local bc = colors[i]
	write('\t%d:' .. align(bc) .. '%q,\n', bc.Number, bc.Name)
end
write('}\n')

write('\n')

write('var brickColorColors = map[BrickColor]Color3{\n')
local format = '\t%%d:%sColor3{%%.%df, %%.%df, %%.%df}, // %%3g, %%3g, %%3g\n'
for i = 1, #colors do
	local bc = colors[i]
	local c = bc.Color
	local f = string.format(format, align(bc), PREC, PREC, PREC)
	write(f, bc.Number, c.r, c.g, c.b, c.r*255, c.g*255, c.b*255)
end
write('}\n')

write('\n')

write('var brickColorPalette = [...]BrickColor{\n')
local i = 0
while true do
	local s, bc = pcall(BrickColor.palette, i)
	if not s then
		break
	end

	if i%8 == 0 then
		write('\t')
	end

	write('%d, ', bc.Number)

	i = i + 1

	if i%8 == 0 then
		write('\n')
	end
end
write('}\n')

msg('begin generated data')
print(data)
msg('end generated data')
