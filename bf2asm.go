package main

import "fmt"
import "os"
import "bufio"

type State struct {
	Loop int
}

type NodeType int

type Node interface {
	Type() NodeType
	Visit(*State)
}

func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeMoveCell NodeType = iota
	NodeAdd
	NodePrint
	NodeGet
	NodeLoop
	NodeList
	NodeClear
)

type ClearNode struct {
	NodeType
	Offset int
}

func newClearNode() *ClearNode {
	return &ClearNode{NodeType: NodeClear}
}

func (n *ClearNode) Visit(st *State) {
	if n.Offset != 0 {
		fmt.Printf("	mov BYTE [rsp+%d], 0 ; Clear Node\n", n.Offset)
	} else {
		fmt.Printf("	mov BYTE [rsp], 0 ; Clear Node\n")
	}
}

type MoveCellNode struct {
	NodeType
	Num int
}

func newMoveCellNode(x int) *MoveCellNode {
	return &MoveCellNode{NodeType: NodeMoveCell, Num: x}
}
func (n *MoveCellNode) Visit(st *State) {
	if n.Num > 0 {
		fmt.Printf("	add rsp, %d ; Next Cell\n", n.Num)
	} else if n.Num < 0 {
		fmt.Printf("	sub rsp, %d ; Prev Cell\n", n.Num*-1)
	}
}

type AddNode struct {
	NodeType
	Num    int
	Offset int
}

func newAddNode(x int) *AddNode {
	return &AddNode{NodeType: NodeAdd, Num: x, Offset: 0}
}

func (n *AddNode) Visit(st *State) {
	if n.Offset != 0 {
		if n.Num > 0 {
			fmt.Printf("	add BYTE [rsp+%d], %d ; Inc Node\n", n.Offset, n.Num)
		} else if n.Num < 0 {
			fmt.Printf("	sub BYTE [rsp+%d], %d ; Dec Node\n", n.Offset, n.Num*-1)
		}
	} else {
		if n.Num > 0 {
			fmt.Printf("	add BYTE [rsp], %d ; Inc Node\n", n.Num)
		} else if n.Num < 0 {
			fmt.Printf("	sub BYTE [rsp], %d ; Dec Node\n", n.Num*-1)
		}
	}
}

type PrintNode struct {
	NodeType
}

func newPrintNode() *PrintNode {
	return &PrintNode{NodeType: NodePrint}
}
func (n *PrintNode) Visit(st *State) {
	// sys_write(stream, message, length)
	fmt.Println("	mov rax, 1")   // sys_write
	fmt.Println("	mov rdi, 1")   // stdout
	fmt.Println("	mov rsi, rsp") // rsp is the data pointer
	fmt.Println("	mov rdx, 1")   // 1 character
	fmt.Println("	syscall")
}

type GetNode struct {
	NodeType
}

func newGetNode() *GetNode {
	return &GetNode{NodeType: NodeGet}
}
func (n *GetNode) Visit(st *State) {
	// sys_read(stream, message, length)
	fmt.Println("	mov rax, 0")   // sys_read
	fmt.Println("	mov rdi, 0")   // stdin
	fmt.Println("	mov rsi, rsp") // rsp is the data pointer
	fmt.Println("	mov rdx, 1")   // 1 character
	fmt.Println("	syscall")
}

type ListNode struct {
	NodeType
	Nodes  []Node
	Parent *ListNode
}

func newListNode(p *ListNode) *ListNode {
	return &ListNode{NodeType: NodeList, Parent: p}
}

func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *ListNode) Visit(st *State) {
	if l.Nodes == nil {
		return
	}
	for _, n := range l.Nodes {
		switch n.Type() {
		case NodeMoveCell:
			n.(*MoveCellNode).Visit(st)
		case NodeAdd:
			n.(*AddNode).Visit(st)
		case NodePrint:
			n.(*PrintNode).Visit(st)
		case NodeGet:
			n.(*GetNode).Visit(st)
		case NodeLoop:
			n.(*LoopNode).Visit(st)
		case NodeList:
			n.(*ListNode).Visit(st)
		case NodeClear:
			n.(*ClearNode).Visit(st)
		}
	}
}

type LoopNode struct {
	NodeType
	Children *ListNode
}

func newLoopNode(l *ListNode) *LoopNode {
	return &LoopNode{NodeType: NodeLoop, Children: l}
}

func (n *LoopNode) Visit(st *State) {

	st.Loop++
	loop_num := st.Loop

	fmt.Println("	cmp BYTE [rsp], 0")
	fmt.Printf("	je loop_end_%d\n", loop_num)
	fmt.Printf("loop_start_%d:\n", loop_num)
	n.Children.Visit(st)
	fmt.Println("	cmp BYTE [rsp], 0")
	fmt.Printf("	jne loop_start_%d\n", loop_num)
	fmt.Printf("loop_end_%d:\n", loop_num)
}

// RLE encoding, [-] to CLEAR
func (l *ListNode) Optimize1(st *State) {
	newList := newListNode(l.Parent)

	if l.Nodes == nil {
		return
	}

	var lastType NodeType = -1
	var lastNode Node = nil

	for _, n := range l.Nodes {
		curType := n.Type()

		if curType == NodeLoop {
			if lastNode != nil {
				newList.append(lastNode)
				lastNode = nil
			}
			n.(*LoopNode).Children.Optimize1(st)

			// Optimize [-] to CLEAR
			if len(n.(*LoopNode).Children.Nodes) == 1 && n.(*LoopNode).Children.Nodes[0].Type() == NodeAdd && n.(*LoopNode).Children.Nodes[0].(*AddNode).Num == -1 {
				lastType = NodeClear
				lastNode = newClearNode()
			} else {
				newList.append(n)
			}
		} else if lastNode == nil {
			lastNode = n
			lastType = curType
		} else if curType == lastType { // RLE optimization
			switch curType {
			case NodeMoveCell:
				lastNode.(*MoveCellNode).Num += n.(*MoveCellNode).Num
			case NodeAdd:
				lastNode.(*AddNode).Num += n.(*AddNode).Num
			case NodeClear:
			default:
				newList.append(lastNode)
				newList.append(n)
				lastNode = nil
				lastType = -1
			}
		} else {
			newList.append(lastNode)
			lastNode = n
			lastType = curType
		}

	}

	if lastNode != nil {
		newList.append(lastNode)
	}

	l.Nodes = newList.Nodes
	return
}

func (l *ListNode) Optimize2(st *State) {

	newList := newListNode(l.Parent)

	if l.Nodes == nil {
		return
	}

	var movCellOffset int = 0

	for _, n := range l.Nodes {
		curType := n.Type()

		if movCellOffset != 0 && (curType == NodePrint || curType == NodeGet) {
			newNode := newMoveCellNode(movCellOffset)
			newList.append(newNode)
			movCellOffset = 0
		}

		if curType == NodeMoveCell {
			movCellOffset += n.(*MoveCellNode).Num
			continue
		} else if curType == NodeAdd {
			// change to MADD
			n.(*AddNode).Offset += movCellOffset
		} else if curType == NodeClear {
			// change to MADD
			n.(*ClearNode).Offset += movCellOffset
		} else if curType == NodeLoop {
			if movCellOffset != 0 {
				newNode := newMoveCellNode(movCellOffset)
				newList.append(newNode)
				movCellOffset = 0
			}
			n.(*LoopNode).Children.Optimize2(st)
		}
		newList.append(n)
	}

	// Restore the original
	if movCellOffset != 0 {
		newNode := newMoveCellNode(movCellOffset)
		newList.append(newNode)
		movCellOffset = 0
	}

	l.Nodes = newList.Nodes
	return
}

func main() {
	if len(os.Args) < 2 {
		panic("You must specify a file")
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("error opening file ", err)
		os.Exit(1)
	}

	fmt.Println("section .text")
	fmt.Println("	global _start")
	fmt.Println("_start:")
	fmt.Println("	push rbp")
	fmt.Println("	mov rbp, rsp")
	fmt.Println("	sub rsp, 65536")
	defer f.Close()
	r := bufio.NewReader(f)
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanBytes)
	list := newListNode(nil)
	startList := list

	var st State
	for scanner.Scan() {
		token := scanner.Bytes()

		switch token[0] {
		case '>':
			node := newMoveCellNode(1)
			list.append(node)
		case '<':
			node := newMoveCellNode(-1)
			list.append(node)
		case '+':
			node := newAddNode(1)
			list.append(node)
		case '-':
			node := newAddNode(-1)
			list.append(node)
		case '.':
			node := newPrintNode()
			list.append(node)
		case ',':
			node := newGetNode()
			list.append(node)
		case '[':
			newList := newListNode(list)
			node := newLoopNode(newList)
			list.append(node)
			list = node.Children

		case ']':
			list = list.Parent
		}
	}

	startList.Optimize1(&st)
	startList.Optimize2(&st)

	startList.Visit(&st)

	fmt.Println("	add rsp, 65536")
	fmt.Println("	pop rbp")
	fmt.Println("	mov rax, 60") // sys_exit
	fmt.Println("	mov rdi, 0")  // exit code 0
	fmt.Println("	syscall")
}
