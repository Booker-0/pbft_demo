# README

This demo is a code implementation of the pbft consensus algorithm.



For the pbft algorithm, because the pbft algorithm needs to support fault-tolerant nodes in addition to fault-tolerant nodes, it also needs to support fault-tolerant evil nodes. Suppose the number of cluster nodes is N and the number of faulty nodes is f. Among the faulty nodes, there can be both faulty nodes and evil nodes, or just faulty nodes or just evil nodes. Then the following two extreme cases arise:

In the first case, the f problematic nodes are both faulty and evil nodes, then according to the principle of fractional majority, the normal nodes in the cluster only need one more node than the f nodes, i.e., f+1 nodes, and the number of sure nodes will be more than the number of faulty nodes, then the cluster can reach consensus. This means that the maximum number of fault-tolerant nodes supported in this case is (n-1)/2.
In the second case, both the faulty nodes and the evil nodes are different nodes. Then there will be f problem nodes and f faulty nodes. When the node is found to be a problem node, it will be excluded from the cluster, leaving f faulty nodes, then according to the principle of fractional majority, the normal nodes in the cluster only need one more node than f nodes, i.e., f+1 nodes, and the number of sure nodes will be more than the number of faulty nodes, then the cluster can reach consensus. So, the number of nodes of all types adds up to f+1 correct nodes, f faulty nodes and f problematic nodes, i.e., 3f+1=n.
Combining these two cases, the maximum number of fault-tolerant nodes supported by the pbft algorithm is therefore (n-1)/3.



## Basic flow of the algorithm



The basic flow of the pbft algorithm consists of the following four steps:

1.The client sends a request to the master node 
2.The master node broadcasts the request to other nodes, and the nodes execute the three-stage consensus process of the pbft algorithm.
3.After processing the three-stage process, the node returns a message to the client.
4.When the client receives the same message from f+1 node, it means that consensus has been completed correctly.
Why does consensus complete correctly when the same message is received from f+1 nodes? From the derivation in the previous section, it is clear that in both the best case and worst case scenarios, if the client receives the same message from f+1 nodes, then there are enough correct nodes that have all reached consensus and completed processing.



## Core three-stage consensus process



1. Pre-prepare phase: When a node receives a pre-prepare message, it has two choices, one is to accept it and the other is not to accept it. When to not accept the pre-prepare message from the master node? A typical case is that if a node receives a pre-pre message with v and n that have appeared in previous messages, but d and m are inconsistent with the previous message or the request number is not between the high and low levels (the concept of high and low levels is explained below), it will reject the request. The logic of rejection is that the master node will not send two messages with the same v and n, but different d and m.

2. Prepare phase: The node agrees to the request and sends prepare messages to other nodes. Note here that not only one node is doing this process at the same time, but there may be n nodes doing this process as well. Therefore, it is possible for a node to receive prepare messages from other nodes. If more than 2f prepare messages are received from different nodes within a certain time frame, it means that the prepare phase is completed.

3. Commit phase: Then it enters the commit phase. The commit message is broadcast to other nodes, and by the same token, this process may be carried out by n nodes. Therefore, it may receive commit messages from other nodes. When it receives 2f+1 commit messages (including itself), it means that most of the nodes have entered the commit phase, and this phase has reached consensus, so the node will execute the request and write data.

  After processing, the node returns a message to the client, which is the entire process of the pbft algorithm