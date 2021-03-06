package actor

//static
func newActorContext(parent SpawnerContext, opts *Options) actorContext {
	ctx := actorContext{
		options:  opts,
		PIDGroup: NewTree(parent),
		UntypeContext: UntypeContext{
			parent: parent.Self(),
			system: parent.System(),
		},
	}
	return ctx
}

//class context (context相当于一个快递公司)
type actorContext struct {
	UntypeContext
	//childs
	PIDGroup
	//New init
	options *Options
	//The real context
	extras *actorContextExtras
}

func (this *actorContext) ActorOf(opts ...Option) ActorRef {
	return NewOptions(opts...).spawn(this)
}

//interface Broker
func (this *actorContext) PreStart() {
	this.incarnateActor()
	//会执行到链路
	this.DispatchMessage(started)
}

func (this *actorContext) PostStop() {
	//关闭自身进程
	this.Stop(this.self)
	//调度自己移除
	this.DispatchMessage(stopped)
}

func (this *actorContext) ThrowFailure(err error, body interface{}) {
	this.DispatchMessage(Failure{err, body})
}

func (this *actorContext) DispatchMessage(body interface{}) {
	switch fn := body.(type) {
	case func():
		fn() //Function is executed directly
	default:
		this.recvMessage(body) //Receive the chain
	}
}

//recvPart
func (this *actorContext) Receive(body MessageEnvelope) {
	this.setMessageEnvelope(body)
	this.Actor().Receive(this.ensureExtras().Context())
	this.setMessageEnvelope(nil)
}

//sendPart
func (this *actorContext) Respond(v interface{}) error {
	return this.Send(this.Sender(), v)
}

func (this *actorContext) Forward(p ActorRef) error {
	return this.Send(p, this.getMessageEnvelope())
}

func (this *actorContext) Request(p ActorRef, v interface{}) error {
	return this.Send(p, this.getSignatureEnvelope(v))
}

func (this *actorContext) Bubble(v interface{}) error {
	return this.Request(this.parent, v)
}

func (this *actorContext) Send(p ActorRef, v interface{}) error {
	return this.sendMessage(p, v)
}

func (this *actorContext) Refuse() error {
	return this.stopIdelActive(this.Sender())
}

//private
func (this *actorContext) sendMessage(sender ActorRef, data interface{}) (err error) {
	if sender == nil {
		err = SendNilErr
	} else {
		//sender the chain
		if sendChain := this.options.senderMiddlewareChain; sendChain != nil {
			err = sendChain(this.ensureExtras().Context(), sender, WrapEnvelope(data))
		} else {
			err = sender.Tell(data)
		}
	}
	return
}

func (this *actorContext) recvMessage(data interface{}) {
	//reader the chain
	if readChain := this.options.receiverMiddlewareChain; readChain != nil {
		readChain(this.ensureExtras().Context(), WrapEnvelope(data))
	} else {
		this.Receive(WrapEnvelope(data))
	}
}

func (this *actorContext) incarnateActor() {
	this.actor = this.options.NewActor()
}

func (this *actorContext) ensureExtras() *actorContextExtras {
	if this.extras == nil {
		this.extras = newActorContextExtras(this.options.ContextWrapper(this))
	}
	return this.extras
}
