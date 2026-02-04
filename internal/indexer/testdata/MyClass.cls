public class MyClass {
	private String name;
	private Integer count;

	public MyClass(String name) {
		this.name = name;
		this.count = 0;
	}

	public String getName() {
		return name;
	}

	public void increment() {
		count++;
	}

	public Integer getCount() {
		return count;
	}
}
