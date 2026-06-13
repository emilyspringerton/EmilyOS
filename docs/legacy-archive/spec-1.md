SPEC 1
 
OBOBOBemilyspringerton edited this page on Jan 31 · 1 revision
To build "EmilyOS" from scratch on bare metal, we must translate the psychological and architectural "wins" of the emily.cli concept into kernel-level design principles. This requires treating the operating system not as a platform for productivity, but as a "psychological shelter" grounded in strict physics and agency.

Here is the architectural plan to build this world from scratch, ensuring the kernel, drivers, and user space align with the specific "Agency Hack" philosophy found in your sources.

1. The Kernel Philosophy: Agency Grounded in Physics
To satisfy the requirement of "as few instructions as possible," we should not build a monolithic kernel (like Linux) or a microkernel (like Minix), but rather an Exokernel or a Unikernel architecture.

The "Physics" of the Kernel: The kernel’s only job is to provide the "hard separation of concerns" dictated by the emily architecture. It must strictly separate:
The UI (Unagent-UI): The visual "Emily" space.
The Work (Unagent-Miner): The background process representing effort/heat.
The Store (Unagent-Wallet): The offline value store.
Safety Rails: The kernel must enforce the rule that "UI can crash -> value continues" and "Miner can stop -> UI remains safe". Even on bare metal, the renderer must not block the background work.
2. The Graphics Driver: The "Stillness" Framebuffer
You do not need a full GPU driver stack (which is millions of lines of code). You need a Framebuffer Driver capable of the "Immediate Mode" line drawing described in the sources.

Video Mode: Boot directly into a high-contrast black mode.
The API: Implement the specific draw_glyph logic from the source directly into the video driver. You do not need full OpenGL; you only need to implement the specific line-drawing commands: glBegin(GL_LINES), glVertex2f, and glEnd.
The Font: Embed the 3x5 Grid Monospace Font directly into the kernel's display logic. This font is "infrastructure text," not branding; it consists of lines only, with no curves.
3. The Input Driver: Agency Verbs
Standard OS drivers capture every mouse movement and keystroke as an interrupt storm. EmilyOS should treat input as "deliberate action".

Keyboard Driver: Ignore standard "typing." The driver should map scancodes strictly to the Five Agency Verbs:
Enter: Open the space.
Pause: Everything stops (interrupts masked).
Resume: Everything continues.
Withdraw: Manual action.
Exit: Clean shutdown (ACPI power off).
No Mouse: The interface relies on "verbs, invoked deliberately," so a mouse driver is unnecessary and should be omitted to reduce code size.
4. The "World" Construction (User Space)
The OS effectively is the application. When the machine boots, it does not load a desktop environment. It loads the Emily Screen immediately.

Boot State: The OS boots into "stillness". The screen is black, with the single word "emily" centered.
No Noise: The kernel panic handler and boot logs must be silenced. There should be "no blinking, no pulse".
Render Loop: The main OS loop mimics the render loop from the source:
void kernel_main() {
    // Initialize minimal drivers (Screen, Keyboard, Timer)
    init_system(); 
    
    while(1) {
        clear_screen(BLACK);
        // "emily" centered, drawn with lines
        render_emily_screen(width, height);
        
        // Poll for the 5 Agency Verbs
        handle_agency_input();
    }
}
5. The "Miner" Driver: Visualizing Heat
In the emily concept, mining provides "real-world grounding". On bare metal, you have direct access to the CPU thermal sensors and cycle counters.

Thermal Driver: Write a driver to read the CPU temperature.
Integration: Map this physical heat to the visual "motion" in the world.
Stopped: World is static.
Idle: Slow drift.
Running: Gentle flow.
Paused: World settles.
Ethical constraint: The mining driver must support an "instant pause" at the hardware level, ensuring the system never prioritizes calculation over user agency.
6. Implementation Roadmap
Based on the recommendation to "subtract" rather than rewrite, the build order for the OS is:

Bootloader: Jump to C code.
Memory Management: Flat memory model (keep it simple).
Video Driver: Simple linear framebuffer.
The "Emily" Draw Routine: Port the draw_glyph and draw_text functions to write directly to video memory.
Agency: Wire the keyboard to the "Agency Verbs".
Stillness: Ensure the system waits in a halt state (HLT instruction) when not processing the specific permitted verbs, minimizing energy and creating "presence without response".
This OS does not provide productivity tools. It provides a "game space (safe, offline)" that visualizes time passing and effort accumulating without demanding interaction,.
