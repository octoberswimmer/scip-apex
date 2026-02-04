public class Greeter {
	private MyClass target;

	public Greeter(MyClass target) {
		this.target = target;
	}

	public String greet() {
		return 'Hello, ' + target.getName();
	}
}
