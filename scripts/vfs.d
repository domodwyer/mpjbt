#!/usr/sbin/dtrace -s

/* 
	vfs.d

	Collects VFS latency measurements and write sizes for the given process
	name.

	Usage: ./vfs.d <execname>
 */

#pragma D option quiet
#pragma D option bufsize=8m
#pragma D option switchrate=10hz
#pragma D option dynvarsize=16m

vfs::vop_read:entry, vfs::vop_write:entry
/ execname == $$1 /
{
	self->ts[stackdepth] = timestamp;
	this->size = args[1]->a_uio->uio_resid;
	this->name = probefunc == "vop_read" ? "read" : "write";
	@iosize1[execname, this->name] = quantize(this->size);
}

vfs::vop_read:return, vfs::vop_write:entry
/this->ts = self->ts[stackdepth]/
{
	this->name = probefunc == "vop_read" ? "read" : "write";
	@lat1[execname, this->name] = quantize(timestamp - this->ts);
	self->ts[stackdepth] = 0;
}

tick-1s
{
	system("date +\"%%Y/%%m/%%d %%H:%%M:%%S\"");
	printa("Latency: %s %s\n%@d\n", @lat1);
	printa("Bytes: %s %s\n%@d\n", @iosize1);
	printf("--------------------------------------------\n\n");
	trunc(@lat1);
	trunc(@iosize1);
}