public class Worker implements Callable {
	public void execute() {
		MyClass mc = new MyClass('test');
		mc.increment();
		String name = mc.getName();
	}
}
