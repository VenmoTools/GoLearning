# TCMalloc
原文链接：http://goog-perftools.sourceforge.net/doc/tcmalloc.html，因为有人翻译了，不再重新翻译，原文：https://www.cnblogs.com/blueoverflow/p/4928369.html

## 引言

TCMalloc要比glibc 2.3的malloc（可以从一个叫作ptmalloc2的独立库获得）和其他我测试过的malloc都快。ptmalloc在一台2.8GHz的P4机器上执行一次小对象malloc及free大约需要300纳秒，而TCMalloc的版本同样的操作大约只需要50纳秒。malloc版本的速度是至关重要的，因为如果malloc不够快，应用程序的作者就倾向于在malloc之上写一个自己的内存释放列表。这就可能导致额外的代码复杂度，以及更多的内存占用――除非作者本身非常仔细地划分释放列表的大小并经常从中清除空闲的对象。

TCMalloc也减少了多线程程序中的锁竞争情况。对于小对象，已经基本上达到了零竞争。对于大对象，TCMalloc尝试使用恰当粒度和有效的自旋锁。ptmalloc同样是通过使用每线程各自的空间来减少锁的竞争，但是ptmalloc2使用每线程空间有一个很大的问题。在ptmalloc2中，内存不可能会从一个空间移动到另一个空间。这有可能导致大量内存被浪费。例如，在一个Google的应用中，第一阶段可能会为其URL标准化的数据结构分配大约300MB内存。当第一阶段结束后，第二阶段将从同样的地址空间开始。如果第二个阶段被安排到了与第一阶段不同的空间内，这个阶段不会复用任何第一阶段留下的的内存，并会给地址空间添加另外一个300MB。类似的内存爆炸问题也可以在其他的应用中看到。

TCMalloc的另一个好处表现在小对象的空间效率。例如，分配N个8字节对象可能要使用大约8N * 1.01字节的空间，即多用百分之一的空间。而ptmalloc2中每个对象都使用了一个四字节的头，我认为并将最终的尺寸圆整为8字节的倍数，最后使用了16N字节。

## 使用
要使用TCMalloc，只要将tcmalloc通过“-ltcmalloc”链接器标志接入你的应用即可。

你也可以通过使用LD_PRELOAD在不是你自己编译的应用中使用tcmalloc：

```
1 $ LD_PRELOAD="/usr/lib/libtcmalloc.so" 
```

LD_PRELOAD比较麻烦，我们也不十分推荐这种用法。

TCMalloc还包含了一个堆检查器(heap checker)以及一个堆测量器(heap profiler )。

如果你更想链接不包含堆测量器和检查器的TCMalloc版本（比如可能为了减少静态二进制文件的大小），你应该链接libtcmalloc_minimal。

## 综述

TCMalloc给每个线程分配了一个线程局部缓存。小对象的分配是直接由线程局部缓存来完成的。如果需要的话会将对象从中央数据结构移动到线程局部缓存中，同时定期的用垃圾收集器把内存从线程局部缓存迁移回中央数据结构中。

TCMalloc将尺寸小于等于32K的对象（“小”对象）和大对象区分开来。大对象直接使用页级分配器（page-level alloctor）（一个页是一个4K的对齐内存区域）从中央堆直接分配。即，一个大对象总是页对齐的并占据了整数个数的页。

连续的一些页面可以被分割为一系列相等大小的小对象。例如，一个连续的页面（4K）可以被划分为32个128字节的对象。

<img src="http://goog-perftools.sourceforge.net/doc/overview.gif" style="text-align:center;">

## 小对象分配

每个小对象的大小都会被映射到与之接近的 60个可分配的尺寸类别中的一个。例如，所有大小在833到1024字节之间的小对象时，都会归整到1024字节。60个可分配的尺寸类别这样隔开：较小的尺寸相差8字节，较大的尺寸相差16字节，再大一点的尺寸差32字节，如此等等。最大的间隔是控制的，这样刚超过上一个级别被分配到下一个级别就不会有太多的内存被浪费。
一个线程缓存包含了由各个尺寸内存的对象组成的单链表，如图所示：

<img src="http://goog-perftools.sourceforge.net/doc/threadheap.gif" style="text-align:center;">

当分配一个小对象时：
+ 我们将其大小映射到对应的尺寸中。  
+ 查找当前线程的线程缓存中相应的尺寸的内存链表。 
+ 如果当前尺寸内存链表非空，那么从链表中移除的第一个对象并返回它。当我们按照这种方式分配时，TCMalloc不需要任何锁。这就可以极大提高分配的速度，因为锁/解锁操作在一个2.8GHz Xeon上大约需要100纳秒的时间。

如果当前尺寸内存链表为空：
+ 从Central Heap中取得一系列这种尺寸的对象（Central Heap是被所有线程共享的）。 
+ 将他们放入该线程线程的缓冲区。 
+ 返回一个新获取的对象给应用程序。

如果Central Heap也为空：
+ 我们从中央页分配器分配了一系列页面。
+ 将他们分割成该尺寸的一系列对象。
+ 将新分配的对象放入Central Heap的链表上 
+ 像前面一样，将部分对象移入线程局部的链表中。

## 大对象分配

一个大对象的尺寸(> 32K)会被中央页堆处理，被圆整到一个页面尺寸（4K）。中央页堆是由 空闲内存列表组成的数组。对于i < 256而言，数组的第k个元素是一个由每个单元是由k个页面组成的空闲内存链表。第256个条目则是一个包含了长度>= 256个页面的空闲内存链表：

k个页面的一次分配通过在第k个空闲内存链表中查找来完成。如果该空闲内存链表为空，那么我们则在下一个空闲内存链表中查找，如此继续。最终，如果必要的话，我们将在最后空闲内存链表中查找。如果这个动作也失败了，我们将向系统获取内存（使用sbrk、mmap或者通过在/dev/mem中进行映射）。

<img src="http://goog-perftools.sourceforge.net/doc/pageheap.gif" style="text-align:center;">

如果k个页面的分配是由连续的> k个页面的空闲内存链表完成的，剩下的连续页面将被重新插回到与之页面大小接近的空闲内存链表中去。

## Span

TCMalloc管理的堆由一系列页面组成。一系列的连续的页面由一个Span对象来表示。一个span可以是已被分配或者是空闲的。如果是空闲的，span 则会是一个页面堆链表中的一个条目。如果已被分配，它会或者是一个已经被传递给应用程序的大对象，或者是一个已经被分割成一系列小对象的一个页面。如果是被分割成小对象的，对象的尺寸类别会被记录在span中。

由页面号索引的中央数组可以用于找到某个页面所属的span对象。例如，下面的span a占据了2个页面，span b占据了1个页面，span c占据了5个页面最后span d占据了3个页面，如图：

<img src="http://goog-perftools.sourceforge.net/doc/spanmap.gif" style="text-align:center;">

在一个32位的地址空间中，中央数组由一个2层的基数树来表示，其中根包含了32个条目，每个叶包含了 215个条目（一个32为地址空间包含了 220个 4K 页面(2^32 / 4k)，一层则是用25整除220个页面)。这就导致了中央阵列的初始内存使用需要128KB空间（215*4字节），看上去还是可以接受的。在64位机器上，我们将使用一个3层的基数树。

## 释放

当一个对象被释放时，我们先计算他的页面号并在中央数组中查找对应的span对象。该span会告诉我们该对象是大是小，如果它是小对象的话尺寸类别是多少。如果是小对象的话，我们将其插入到当前线程的线程缓存中对应的空闲内存链表中。如果线程缓存现在超过了某个预定的大小（默认为2MB），我们便运行垃圾收集器将未使用的对象从线程缓存中移入中央自由列表。

如果该对象是大对象的话，span对象会告诉我们该对象包含的页面的范围。假设该范围是[p,q]。我们还会查找页面p-1和页面q+1对应的span对象。如果这两个相邻的span中有任何一个是空闲的，我们将他们和[p,q]的span接合起来。最后span会被插入到页面堆中合适的空闲链表中。


## 小对象的重要空闲内存链表

就像前面提过的一样，我们为每一个尺寸类别设置了一个中央空闲列表。每个中央空闲列表由两层数据结构来组成：一系列span和每个span对象的一个空闲内存的链表。

一个对象是通过从某个span对象的空闲列表中取出第一个条目来分配的。（如果所有的span里只有空链表，那么首先从中央页面堆中分配一个尺寸合适的span。）

一个对象通过将其添加到它包含的span对象的空闲内存链表中来将还回中央空闲列表。如果链表长度现在等于span对象中所有小对象的数量，那么该span就是完全自由的了，就会被返回到页面堆中（span对象中所有小对象都回收完了，整个span对象就空闲了）。

## 线程缓冲区的垃圾回收

垃圾回收对象保证线程缓冲区的大小可控制并将未使用的对象交还给中央空闲列表。有的线程需要大量的缓冲来保证工作有很好的性能，而有的线程只需要很少甚至不需要缓冲就能工作，当一个线程的缓冲区超过它的max_size，垃圾回收对象介入，之后这个线程就要和其它线程竞争获取更大的缓冲。

垃圾回收仅仅会在内存释放的时候才会允许。我们检查所有的空闲内存链表并把一些数量的对象从空闲列表移动到中央链表。

从某个空闲链表中移除的对象的数量是通过使用一个每空闲链表的低水位线L来确定的。L记录了自上一次垃圾收集以来列表最短的长度。注意，在上一次的垃圾收集中我们可能只是将列表缩短了L个对象而没有对中央列表进行任何额外访问。我们利用这个过去的历史作为对未来访问的预测器并将L/2个对象从线程缓存空闲列表列表中移到相应的中央空闲链表中。这个算法有个很好的特性是，如果某个线程不再使用某个特定的尺寸时，该尺寸的所有对象都会很快从线程缓存被移到中央空闲链表，然后可以被其他缓存利用。

如果在线程中，某个大小的内存对象持续释放比分配操作多，这种2/L行为会引起至少有L/2的对象长期处于空闲链表中，为了避免这种内存浪费，我们减少每个链表的最大长度num_objects_to_move个。

## 性能
### PTMalloc2单元测试
 
PTMalloc2包（现在已经是glibc的一部分了）包含了一个单元测试程序t-test1.c。它会产生一定数量的线程并在每个线程中进行一系列分配和解除分配；线程之间没有任何通信除了在内存分配器中同步。

t-test1（放在tests/tcmalloc/中，编译为ptmalloc_unittest1）用一系列不同的线程数量（1～20）和最大分配尺寸（64B～32KB）运行。这些测试运行在一个2.4GHz 双核心Xeon的RedHat 9系统上，并启用了超线程技术， 使用了Linux glibc-2.3.2，每个测试中进行一百万次操作。在每个案例中，一次正常运行，一次使用LD_PRELOAD=libtcmalloc.so。

下面的图像显示了TCMalloc对比PTMalloc2在不同的衡量指标下的性能。首先，现实每秒操作次数（百万）以及最大分配尺寸，针对不同数量的线程。用来生产这些图像的原始数据（time工具的输出）可以在t-test1.times.txt中找到。

<table>
  <tr>
    <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.1.threads.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.2.threads.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.3.threads.png">
    </td>
  </tr>
  <tr>
    <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.4.threads.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.5.threads.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.8.threads.png">
    </td>
  </tr>
  <tr>
    <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.12.threads.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.16.threads.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspersec.vs.size.20.threads.png">
    </td>
  </tr>
</table>

+ TCMalloc要比PTMalloc2更具有一致地伸缩性——对于所有线程数量>1的测试，小分配达到了约7～9百万操作每秒，大分配降到了约2百万操作每秒。单线程的案例则明显是要被剔除的，因为他只能保持单个处理器繁忙因此只能获得较少的每秒操作数。PTMalloc2在每秒操作数上有更高的方差——某些地方峰值可以在小分配上达到4百万操作每秒，而在大分配上降到了<1百万操作每秒。
+ TCMalloc在绝大多数情况下要比PTMalloc2快，并且特别是小分配上。线程间的争用在TCMalloc中问题不大。
+ TCMalloc的性能随着分配尺寸的增加而降低。这是因为每线程缓存当它达到了阈值（默认是2MB）的时候会被垃圾收集。对于更大的分配尺寸，在垃圾收集之前只能在缓存中存储更少的对象。
+ TCMalloc性能在约32K最大分配尺寸附件有一个明显的下降。这是因为在每线程缓存中的32K对象的最大尺寸；对于大于这个值得对象TCMalloc会从中央页面堆中进行分配。

下面是每秒CPU时间的操作数（百万）以及线程数量的图像，最大分配尺寸64B～128KB。


<table>
  <tr>
    <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.64.bytes.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.256.bytes.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.1024.bytes.png">
    </td>
  </tr>
  <tr>
    <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.4096.bytes.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.8192.bytes.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.16384.bytes.png">
    </td>
  </tr>
  <tr>
    <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.32768.bytes.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.65536.bytes.png">
    </td>
     <td>
      <img src="http://goog-perftools.sourceforge.net/doc/tcmalloc-opspercpusec.vs.threads.131072.bytes.png">
    </td>
  </tr>
</table>
这次我们再一次看到TCMalloc要比PTMalloc2更连续也更高效。对于<32K的最大分配尺寸，TCMalloc在大线程数的情况下典型地达到了CPU时间每秒约0.5～1百万操作，同时PTMalloc通常达到了CPU时间每秒约0.5～1百万，还有很多情况下要比这个数字小很多。在32K最大分配尺寸之上，TCMalloc下降到了每CPU时间秒1～1.5百万操作，同时PTMalloc对于大线程数降到几乎只有零（也就是，使用PTMalloc，在高度多线程的情况下，很多CPU时间被浪费在轮流等待锁定上了）。

## 附加说明

对于某些系统，TCMalloc可能无法与没有链接libpthread.so（或者你的系统上同等的东西）的应用程序正常工作。它应该能正常工作于使用glibc 2.3的Linux上，但是其他OS/libc的组合方式尚未经过任何测试。

TCMalloc可能要比其他malloc版本在某种程度上更吃内存，（但是倾向于不会有其他malloc版本中可能出现的爆发性增长。）尤其是在启动时TCMalloc会分配大约240KB的内部内存。

不要试图将TCMalloc载入到一个运行中的二进制程序中（例如，在Java中使用JNI）。二进制程序已经使用系统malloc分配了一些对象，并会尝试将它们传递到TCMalloc进行解除分配。TCMalloc是无法处理这种对象的。
  
  
  


